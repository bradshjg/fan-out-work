package handlers

import (
	"fmt"

	"github.com/bradshjg/fan-out-work/services"
	"github.com/bradshjg/fan-out-work/views"
	"github.com/labstack/echo/v4"
)

const StopPollingStatus = 286

func NewFanoutHandler(fanoutService services.FanoutService) *FanoutHandler {
	return &FanoutHandler{
		fanoutService: fanoutService,
	}
}

type FanoutHandler struct {
	fanoutService services.FanoutService
}

func (fh *FanoutHandler) HomeHandler(c echo.Context) error {
	_, err := fh.fanoutService.AccessToken(c)
	if err != nil {
		fh.fanoutService.ClearSession(c)
		return renderView(c, views.Index(false, []string{}, []string{}))
	}
	orgs, err := fh.fanoutService.Orgs(c)
	if err != nil {
		return fmt.Errorf("unable to get orgs: %w", err)
	}
	patches, err := fh.fanoutService.Patches()
	if err != nil {
		return fmt.Errorf("error getting patches: %w", err)
	}
	return renderView(c, views.Index(true, orgs, patches))
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
		return fmt.Errorf("invalid request: %w", err)
	}
	token, err := fh.fanoutService.AccessToken(c)
	if err != nil {
		return fmt.Errorf("error getting access token: %w", err)
	}
	pr := services.PatchRun{
		AccessToken: token,
		Org:         patch.Org,
		Patch:       patch.Name,
		DryRun:      patch.DryRun,
	}
	outputToken, err := fh.fanoutService.Run(pr)
	if err != nil {
		return fmt.Errorf("error getting output token: %w", err)
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
		return fmt.Errorf("invalid request: %w", err)
	}
	lines, done, err := fh.fanoutService.Output(output.Token)
	if err != nil {
		return fmt.Errorf("error getting output: %w", err)
	}
	if done {
		c.Response().Writer.WriteHeader(StopPollingStatus) // HTMX handles the semantics here
		return renderView(c, views.Output(lines, output.Org, output.Patch, output.DryRun))
	} else {
		return renderView(c, views.Output(lines, output.Org, output.Patch, false))
	}
}
