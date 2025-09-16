package handlers

import (
	"fmt"
	"net/http"

	"github.com/bradshjg/fan-out-work/services"
	"github.com/bradshjg/fan-out-work/views"
	"github.com/labstack/echo/v4"
)

func NewFanoutHandler(githubService services.GitHubService) *FanoutHandler {
	return &FanoutHandler{
		githubService: githubService,
	}
}

type FanoutHandler struct {
	githubService services.GitHubService
}

func (fh *FanoutHandler) HomeHandler(c echo.Context) error {
	orgs, err := fh.githubService.GetOrgs(c)
	if err != nil {
		fmt.Printf("Error getting orgs: %v", err)
		return c.Redirect(http.StatusFound, "/foo")
	}
	return renderView(c, views.Home(orgs))
}
