package service

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"prmanager/internal/models"
	"prmanager/internal/repository"
	"time"
)

// Service - реализация сервиса
type Service struct {
	repo   repository.Repository
	logger *log.Logger
}

func NewService(repo repository.Repository, logger *log.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// Teams
func (s *Service) CreateTeam(ctx context.Context, team *models.Team) (*models.Team, error) {
	s.logger.Printf("Creating team: %s", team.TeamName)

	// Проверяем существует ли команда
	exists, err := s.repo.TeamExists(ctx, team.TeamName)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("TEAM_EXISTS")
	}

	// Создаем команду
	err = s.repo.CreateTeam(ctx, team)
	if err != nil {
		return nil, err
	}

	return team, nil
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	s.logger.Printf("Getting team: %s", teamName)

	team, err := s.repo.GetTeam(ctx, teamName)
	if err != nil {
		return nil, errors.New("NOT_FOUND")
	}

	return team, nil
}

// Users
func (s *Service) SetUserActive(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	s.logger.Printf("Setting user %s active: %t", userID, isActive)

	user, err := s.repo.UpdateUserActive(ctx, userID, isActive)
	if err != nil {
		return nil, errors.New("NOT_FOUND")
	}

	return user, nil
}

func (s *Service) GetUserReviews(ctx context.Context, userID string) ([]*models.PullRequestShort, error) {
	s.logger.Printf("Getting reviews for user: %s", userID)

	// Проверяем существует ли пользователь
	_, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, errors.New("NOT_FOUND")
	}

	prs, err := s.repo.GetPullRequestsByReviewer(ctx, userID)
	if err != nil {
		return nil, err
	}

	return prs, nil
}

// Pull Requests
func (s *Service) CreatePullRequest(ctx context.Context, prID, title, authorID string) (*models.PullRequest, error) {
	s.logger.Printf("Creating PR: %s, author: %s", prID, authorID)

	// Проверяем существует ли PR
	exists, err := s.repo.PRExists(ctx, prID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("PR_EXISTS")
	}

	// Проверяем существует ли автор
	author, err := s.repo.GetUser(ctx, authorID)
	if err != nil {
		return nil, errors.New("AUTHOR_NOT_FOUND")
	}

	// Получаем активных пользователей команды для назначения ревьюеров
	teamUsers, err := s.repo.GetActiveUsersByTeam(ctx, author.TeamName)
	if err != nil {
		return nil, errors.New("TEAM_NOT_FOUND")
	}

	// Автоназначение ревьюеров
	reviewerIDs := s.autoAssignReviewers(authorID, teamUsers)

	pr := &models.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   title,
		AuthorID:          authorID,
		Status:            "OPEN",
		AssignedReviewers: reviewerIDs,
		CreatedAt:         time.Now(),
	}

	// Создаем PR
	err = s.repo.CreatePullRequest(ctx, pr)
	if err != nil {
		return nil, err
	}

	// Назначаем ревьюеров (если есть)
	if len(reviewerIDs) > 0 {
		err = s.repo.AssignReviewers(ctx, prID, reviewerIDs)
		if err != nil {
			return nil, err
		}
	}

	return pr, nil
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	s.logger.Printf("Merging PR: %s", prID)

	// Получаем PR
	pr, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, errors.New("NOT_FOUND")
	}

	// Если уже мержен - возвращаем как есть (идемпотентность)
	if pr.Status == "MERGED" {
		return pr, nil
	}

	// Обновляем статус
	mergedAt := time.Now()
	updatedPR, err := s.repo.UpdatePullRequestStatus(ctx, prID, "MERGED", &mergedAt)
	if err != nil {
		return nil, err
	}

	return updatedPR, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*models.ReassignResult, error) {
	s.logger.Printf("Reassigning reviewer %s in PR: %s", oldReviewerID, prID)

	// Получаем PR
	pr, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, errors.New("NOT_FOUND")
	}

	// Проверяем что PR не мержен
	if pr.Status == "MERGED" {
		return nil, errors.New("PR_MERGED")
	}

	// Проверяем что старый ревьюер назначен на PR
	isAssigned := false
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer == oldReviewerID {
			isAssigned = true
			break
		}
	}
	if !isAssigned {
		return nil, errors.New("NOT_ASSIGNED")
	}

	// Получаем команду старого ревьюера
	oldReviewer, err := s.repo.GetUser(ctx, oldReviewerID)
	if err != nil {
		return nil, errors.New("NOT_FOUND")
	}

	// Получаем доступных кандидатов из команды
	candidates, err := s.repo.GetActiveUsersByTeam(ctx, oldReviewer.TeamName)
	if err != nil {
		return nil, errors.New("TEAM_NOT_FOUND")
	}

	// Выбираем нового ревьюера
	newReviewerID, err := s.selectNewReviewer(pr, oldReviewerID, candidates)
	if err != nil {
		return nil, errors.New("NO_CANDIDATE")
	}

	// Выполняем переназначение
	err = s.repo.ReassignReviewer(ctx, prID, oldReviewerID, newReviewerID)
	if err != nil {
		return nil, err
	}

	// Получаем обновленный PR
	updatedPR, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, err
	}

	return &models.ReassignResult{
		PR:            updatedPR,
		NewReviewerID: newReviewerID,
	}, nil
}

// Вспомогательные методы
func (s *Service) autoAssignReviewers(authorID string, teamUsers []*models.User) []string {
	var candidates []string
	for _, user := range teamUsers {
		if user.UserID != authorID && user.IsActive {
			candidates = append(candidates, user.UserID)
		}
	}

	if len(candidates) == 0 {
		return []string{}
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	maxReviewers := 2
	if len(candidates) < maxReviewers {
		maxReviewers = len(candidates)
	}

	return candidates[:maxReviewers]
}

func (s *Service) selectNewReviewer(pr *models.PullRequest, oldReviewerID string, candidates []*models.User) (string, error) {
	var availableCandidates []string

	for _, candidate := range candidates {
		if candidate.UserID != pr.AuthorID && candidate.UserID != oldReviewerID {
			isCurrentReviewer := false
			for _, reviewer := range pr.AssignedReviewers {
				if reviewer == candidate.UserID {
					isCurrentReviewer = true
					break
				}
			}

			if !isCurrentReviewer {
				availableCandidates = append(availableCandidates, candidate.UserID)
			}
		}
	}

	if len(availableCandidates) == 0 {
		return "", errors.New("no available candidates")
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	return availableCandidates[rand.Intn(len(availableCandidates))], nil
}
