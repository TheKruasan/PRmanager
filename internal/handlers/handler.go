package handlers

import (
	"log"
	"prmanager/internal/handlers/interfaces"
	prhandler "prmanager/internal/handlers/pr_handler"
	teamhandler "prmanager/internal/handlers/team_handler"
	userhandler "prmanager/internal/handlers/user_handler"
)

type Handler struct {
	TeamHandler        *teamhandler.Handler
	UserHandler        *userhandler.Handler
	PullRequestHandler *prhandler.Handler
}

func NewHandler(service interfaces.Service, logger *log.Logger) *Handler {
	return &Handler{
		TeamHandler:        teamhandler.NewHandler(service, logger),
		UserHandler:        userhandler.NewHandler(service, logger),
		PullRequestHandler: prhandler.NewHandler(service, logger),
	}
}
