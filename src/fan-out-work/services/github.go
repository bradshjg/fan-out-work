package services

import (
	"context"

	"github.com/google/go-github/v74/github"
	"github.com/labstack/echo/v4"
)

func NewGitHubService(oauthService *OAuthService) *GitHubService {
	return &GitHubService{
		oauthService: oauthService,
	}
}

type GitHubService struct {
	oauthService *OAuthService
}

func (gs *GitHubService) GetOrgs(c echo.Context) ([]string, error) {
	ctx := context.Background()
	client, err := gs.oauthService.GetClient(c)
	if err != nil {
		return []string{}, err
	}
	opt := &github.ListOptions{
		PerPage: 100,
	}
	var allOrgs []string
	for {
		orgs, resp, err := client.Organizations.List(ctx, "", opt)
		if err != nil {
			return []string{}, err
		}
		for _, org := range orgs {
			allOrgs = append(allOrgs, org.GetLogin())
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allOrgs, nil
}
