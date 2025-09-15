package handlers

import (
	"net/http"

	"github.com/bradshjg/fan-out-work/services"
	"github.com/labstack/echo/v4"
)

func NewGitHubHandler(oauthService services.OAuthService) *GitHubHandler {
	return &GitHubHandler{
		oauthService: oauthService,
	}
}

type GitHubHandler struct {
	oauthService services.OAuthService
}

func (gh *GitHubHandler) OAuthHandler(c echo.Context) error {
	redirectURL, err := gh.oauthService.RedirectURL(c)
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, redirectURL)
}

func (gh *GitHubHandler) OAuthCallbackHandler(c echo.Context) error {
	gh.oauthService.StoreToken(c)
	return c.Redirect(http.StatusFound, "/")
}
