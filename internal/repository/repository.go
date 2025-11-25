package repository

import (
	"context"
	"fmt"
	"log"
	"prmanager/internal/models"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db     *pgxpool.Pool
	logger *log.Logger
}

func NewRepository(db *pgxpool.Pool, logger *log.Logger) Repository {
	return Repository{
		db:     db,
		logger: logger,
	}
}

// Teams
func (r *Repository) CreateTeam(ctx context.Context, team *models.Team) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Создаем команду
	var teamID string
	err = tx.QueryRow(ctx,
		"INSERT INTO teams (name) VALUES ($1) RETURNING id",
		team.TeamName,
	).Scan(&teamID)
	if err != nil {
		return fmt.Errorf("insert team: %w", err)
	}

	for _, member := range team.Members {
		var existingUserID string
		err := tx.QueryRow(ctx,
			"SELECT id FROM users WHERE id = $1",
			member.UserID,
		).Scan(&existingUserID)

		switch err {
		case pgx.ErrNoRows:
			_, err = tx.Exec(ctx,
				"INSERT INTO users (id, username, team_id, is_active) VALUES ($1, $2, $3, $4)",
				member.UserID, member.Username, teamID, member.IsActive,
			)
		case nil:
			_, err = tx.Exec(ctx,
				"UPDATE users SET username = $1, team_id = $2, is_active = $3 WHERE id = $4",
				member.Username, teamID, member.IsActive, member.UserID,
			)
		}

		if err != nil {
			return fmt.Errorf("upsert user %s: %w", member.UserID, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	var team models.Team
	team.TeamName = teamName

	rows, err := r.db.Query(ctx,
		`SELECT u.id, u.username, u.is_active 
		 FROM users u 
		 JOIN teams t ON u.team_id = t.id 
		 WHERE t.name = $1`,
		teamName,
	)
	if err != nil {
		return nil, fmt.Errorf("query team members: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.UserID, &user.Username, &user.IsActive)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		user.TeamName = teamName
		team.Members = append(team.Members, user)
	}

	if len(team.Members) == 0 {
		return nil, fmt.Errorf("team not found")
	}

	return &team, nil
}

func (r *Repository) TeamExists(ctx context.Context, teamName string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM teams WHERE name = $1)",
		teamName,
	).Scan(&exists)
	return exists, err
}

func (r *Repository) GetTeamByName(ctx context.Context, teamName string) (*models.Team, error) {
	return r.GetTeam(ctx, teamName)
}

// Users
func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	// Находим team_id по team_name
	var teamID string
	err := r.db.QueryRow(ctx,
		"SELECT id FROM teams WHERE name = $1",
		user.TeamName,
	).Scan(&teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	_, err = r.db.Exec(ctx,
		"INSERT INTO users (id, username, team_id, is_active) VALUES ($1, $2, $3, $4)",
		user.UserID, user.Username, teamID, user.IsActive,
	)
	return err
}

func (r *Repository) GetUser(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	var teamName string

	err := r.db.QueryRow(ctx,
		`SELECT u.id, u.username, u.is_active, t.name 
		 FROM users u 
		 JOIN teams t ON u.team_id = t.id 
		 WHERE u.id = $1`,
		userID,
	).Scan(&user.UserID, &user.Username, &user.IsActive, &teamName)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("query user: %w", err)
	}

	user.TeamName = teamName
	return &user, nil
}

func (r *Repository) UpdateUser(ctx context.Context, user *models.User) error {
	// Находим team_id по team_name
	var teamID string
	err := r.db.QueryRow(ctx,
		"SELECT id FROM teams WHERE name = $1",
		user.TeamName,
	).Scan(&teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	_, err = r.db.Exec(ctx,
		"UPDATE users SET username = $1, team_id = $2, is_active = $3 WHERE id = $4",
		user.Username, teamID, user.IsActive, user.UserID,
	)
	return err
}

func (r *Repository) UpdateUserActive(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	_, err := r.db.Exec(ctx,
		"UPDATE users SET is_active = $1 WHERE id = $2",
		isActive, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("update user active: %w", err)
	}

	return r.GetUser(ctx, userID)
}

func (r *Repository) GetActiveUsersByTeam(ctx context.Context, teamName string) ([]*models.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT u.id, u.username, u.is_active 
		 FROM users u 
		 JOIN teams t ON u.team_id = t.id 
		 WHERE t.name = $1 AND u.is_active = true`,
		teamName,
	)
	if err != nil {
		return nil, fmt.Errorf("query active users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.UserID, &user.Username, &user.IsActive)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		user.TeamName = teamName
		users = append(users, &user)
	}

	return users, nil
}

func (r *Repository) GetUsersByTeamName(ctx context.Context, teamName string) ([]*models.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT u.id, u.username, u.is_active 
		 FROM users u 
		 JOIN teams t ON u.team_id = t.id 
		 WHERE t.name = $1`,
		teamName,
	)
	if err != nil {
		return nil, fmt.Errorf("query users by team name: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.UserID, &user.Username, &user.IsActive)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		user.TeamName = teamName
		users = append(users, &user)
	}

	return users, nil
}

// Pull Requests
func (r *Repository) CreatePullRequest(ctx context.Context, pr *models.PullRequest) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Создаем PR
	_, err = tx.Exec(ctx,
		"INSERT INTO pull_requests (id, title, author_id, status) VALUES ($1, $2, $3, $4)",
		pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status,
	)
	if err != nil {
		return fmt.Errorf("insert pull request: %w", err)
	}

	// Назначаем ревьюеров
	for _, reviewerID := range pr.AssignedReviewers {
		_, err = tx.Exec(ctx,
			"INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)",
			pr.PullRequestID, reviewerID,
		)
		if err != nil {
			return fmt.Errorf("assign reviewer %s: %w", reviewerID, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetPullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	var createdAt time.Time
	var mergedAt *time.Time

	// Получаем основную информацию о PR
	err := r.db.QueryRow(ctx,
		`SELECT id, title, author_id, status, created_at, merged_at 
		 FROM pull_requests 
		 WHERE id = $1`,
		prID,
	).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("pull request not found")
		}
		return nil, fmt.Errorf("query pull request: %w", err)
	}

	pr.CreatedAt = createdAt
	pr.MergedAt = mergedAt

	// Получаем назначенных ревьюеров
	rows, err := r.db.Query(ctx,
		"SELECT user_id FROM pr_reviewers WHERE pr_id = $1",
		prID,
	)
	if err != nil {
		return nil, fmt.Errorf("query reviewers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var reviewerID string
		err := rows.Scan(&reviewerID)
		if err != nil {
			return nil, fmt.Errorf("scan reviewer: %w", err)
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
	}

	return &pr, nil
}

func (r *Repository) UpdatePullRequestStatus(ctx context.Context, prID, status string, mergedAt *time.Time) (*models.PullRequest, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if mergedAt != nil {
		_, err = tx.Exec(ctx,
			"UPDATE pull_requests SET status = $1, merged_at = $2 WHERE id = $3",
			status, mergedAt, prID,
		)
	} else {
		_, err = tx.Exec(ctx,
			"UPDATE pull_requests SET status = $1 WHERE id = $2",
			status, prID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("update pull request status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetPullRequest(ctx, prID)
}

func (r *Repository) GetPullRequestsByReviewer(ctx context.Context, userID string) ([]*models.PullRequestShort, error) {
	rows, err := r.db.Query(ctx,
		`SELECT pr.id, pr.title, pr.author_id, pr.status
		 FROM pull_requests pr
		 JOIN pr_reviewers prr ON pr.id = prr.pr_id
		 WHERE prr.user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query pull requests by reviewer: %w", err)
	}
	defer rows.Close()

	var prs []*models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status)
		if err != nil {
			return nil, fmt.Errorf("scan pull request: %w", err)
		}
		prs = append(prs, &pr)
	}

	return prs, nil
}

func (r *Repository) AssignReviewers(ctx context.Context, prID string, reviewerIDs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, reviewerID := range reviewerIDs {
		_, err = tx.Exec(ctx,
			"INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)",
			prID, reviewerID,
		)
		if err != nil {
			return fmt.Errorf("assign reviewer %s: %w", reviewerID, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) ReassignReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Удаляем старого ревьюера
	_, err = tx.Exec(ctx,
		"DELETE FROM pr_reviewers WHERE pr_id = $1 AND user_id = $2",
		prID, oldReviewerID,
	)
	if err != nil {
		return fmt.Errorf("remove old reviewer: %w", err)
	}

	// Добавляем нового ревьюера
	_, err = tx.Exec(ctx,
		"INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)",
		prID, newReviewerID,
	)
	if err != nil {
		return fmt.Errorf("add new reviewer: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *Repository) PRExists(ctx context.Context, prID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pull_requests WHERE id = $1)",
		prID,
	).Scan(&exists)
	return exists, err
}
