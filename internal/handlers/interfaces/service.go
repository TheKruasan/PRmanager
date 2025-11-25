package interfaces

import (
	"context"
	"prmanager/internal/models"
)

type Service interface {
	// Teams
	CreateTeam(ctx context.Context, team *models.Team) (*models.Team, error)
	GetTeam(ctx context.Context, teamName string) (*models.Team, error)

	// Users
	SetUserActive(ctx context.Context, userID string, isActive bool) (*models.User, error)
	GetUserReviews(ctx context.Context, userID string) ([]*models.PullRequestShort, error)

	// Pull Requests
	CreatePullRequest(ctx context.Context, prID, title, authorID string) (*models.PullRequest, error)
	MergePullRequest(ctx context.Context, prID string) (*models.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*models.ReassignResult, error)
}
