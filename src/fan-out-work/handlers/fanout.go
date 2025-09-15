package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradshjg/fan-out-work/services"
	"github.com/bradshjg/fan-out-work/views"
	"github.com/labstack/echo/v4"
)

func NewFanoutHandler(oauthService services.OAuthService) *FanoutHandler {
	return &FanoutHandler{
		oauthService: &oauthService,
	}
}

type FanoutHandler struct {
	oauthService *services.OAuthService
}

func (fh *FanoutHandler) HomeHandler(c echo.Context) error {
	ctx := context.Background()
	client, err := fh.oauthService.GetClient(c)
	if err != nil {
		return c.Redirect(http.StatusFound, "/foo")
	}
	user, resp, err := client.Users.Get(ctx, "")
	fmt.Printf("%v", resp.StatusCode)
	if err != nil {
		return c.Redirect(http.StatusFound, "/foo")
	}
	name := user.GetName()
	return renderView(c, views.Home(name))
}
