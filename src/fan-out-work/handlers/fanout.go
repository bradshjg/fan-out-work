package handlers

import (
	"fmt"
	"net/http"

	"github.com/bradshjg/fan-out-work/services"
	"github.com/bradshjg/fan-out-work/views"
	"github.com/labstack/echo/v4"
)

const StopPollingStatus = 286

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
	orgs, err := fh.githubService.Orgs(c)
	authenticated := err == nil
	patches, err := fh.fanoutService.Patches()
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error getting patches: %v", err))
	}
	return renderView(c, views.Index(authenticated, orgs, patches))
}

type Patch struct {
	Org    string `form:"org"`
	Name   string `form:"patch"`
	DryRun bool   `form:"dry-run"`
}

func (fh *FanoutHandler) RunHandler(c echo.Context) error {
	patch := new(Patch)
	err := c.Bind(patch)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
	}
	token, err := fh.githubService.AccessToken(c)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error getting access token: %v", err))
	}
	pr := services.PatchRun{
		AccessToken: token,
		Org:         patch.Org,
		Patch:       patch.Name,
		DryRun:      patch.DryRun,
	}
	outputToken, err := fh.fanoutService.Run(pr)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error getting output token: %v", err))
	}
	return renderView(c, views.Run(outputToken, patch.Org, patch.Name, patch.DryRun))
}

type Output struct {
	Org    string `query:"org"`
	Patch  string `query:"patch"`
	DryRun bool   `query:"dry-run"`
	Token  string `query:"token"`
}

func (fh *FanoutHandler) OutputHandler(c echo.Context) error {
	var output Output
	err := c.Bind(&output)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
	}
	lines, done, err := fh.fanoutService.Output(output.Token)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error getting output: %v", err))
	}
	if done {
		c.Response().Writer.WriteHeader(StopPollingStatus) // HTMX handles the semantics here
		return renderView(c, views.Output(lines, output.Org, output.Patch, output.DryRun))
	} else {
		return renderView(c, views.Output(lines, output.Org, output.Patch, false))
	}
}
