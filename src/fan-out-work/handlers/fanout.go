package handlers

import (
	"github.com/bradshjg/fan-out-work/services"
	"github.com/bradshjg/fan-out-work/views"
	"github.com/labstack/echo/v4"
)

func NewFanoutHandler(githubService services.GitHubService, fanoutService services.FanoutService) *FanoutHandler {
	return &FanoutHandler{
		githubService: githubService,
		fanoutService: fanoutService,
	}
}

type FanoutHandler struct {
	githubService services.GitHubService
	fanoutService services.FanoutService
}

func (fh *FanoutHandler) HomeHandler(c echo.Context) error {
	orgs, err := fh.githubService.GetOrgs(c)
	authenticated := err == nil
	patches, err := fh.fanoutService.GetPatches()
	if err != nil {
		c.Logger().Error(err)
	}
	return renderView(c, views.Index(authenticated, orgs, patches))
}
