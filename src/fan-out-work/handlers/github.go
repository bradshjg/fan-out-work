package handlers

import (
	"fmt"
	"net/http"

	"github.com/bradshjg/fan-out-work/services"
	"github.com/labstack/echo/v4"
)

func NewGitHubHandler(oauthService *services.OAuthService) *GitHubHandler {
	return &GitHubHandler{
		oauthService: oauthService,
	}
}

type GitHubHandler struct {
	oauthService *services.OAuthService
}

func (gh *GitHubHandler) OAuthHandler(c echo.Context) error {
	redirectURL, err := gh.oauthService.RedirectURL(c)
	if err != nil {
		return fmt.Errorf("error generating redirect url: %w", err)
	}
	return c.Redirect(http.StatusFound, redirectURL)
}

func (gh *GitHubHandler) OAuthCallbackHandler(c echo.Context) error {
	err := gh.oauthService.StoreToken(c)
	if err != nil {
		return fmt.Errorf("error storing token in oauth callback: %w", err)
	}
	return c.Redirect(http.StatusFound, "/")
}
