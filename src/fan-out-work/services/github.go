package services

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-github/v74/github"
	"github.com/labstack/echo/v4"
)

type GitHubService interface {
	ClearSession(c echo.Context)
	Orgs(c echo.Context) ([]string, error)
	GetOrCreateIssue(c echo.Context, i Issue) (string, error)
	AccessToken(c echo.Context) (string, error)
}

func NewGitHubService(oauthService *OAuthService) *GitHubAPIService {
	return &GitHubAPIService{
		oauthService: oauthService,
	}
}

type GitHubAPIService struct {
	oauthService *OAuthService
}

func (gs *GitHubAPIService) ClearSession(c echo.Context) {
	gs.oauthService.ClearSession(c)
}

func (gs *GitHubAPIService) AccessToken(c echo.Context) (string, error) {
	token, err := gs.oauthService.AccessToken(c)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (gs *GitHubAPIService) Orgs(c echo.Context) ([]string, error) {
	ctx := context.Background()
	client, err := gs.oauthService.Client(c)
	if err != nil {
		return []string{}, fmt.Errorf("error getting client: %w", err)
	}
	opt := &github.ListOptions{
		PerPage: 100,
	}
	var allOrgs []string
	for {
		orgs, resp, err := client.Organizations.List(ctx, "", opt)
		if err != nil {
			return []string{}, fmt.Errorf("error listing orgs: %w", err)
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

type Issue struct {
	Owner string
	Body  string
	Title string
}

// Gets or creates a GitHub issue
func (gs *GitHubAPIService) GetOrCreateIssue(c echo.Context, i Issue) (string, error) {
	const fanoutRepo = "fan-out"
	ctx := context.Background()
	client, err := gs.oauthService.Client(c)
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	repo, _, err := client.Repositories.Get(ctx, i.Owner, fanoutRepo)
	if err != nil {
		return "", err
	}
	if repo == nil {
		return "create a \"fan-out\" repository for tracking merges", nil
	}
	opt := &github.IssueListByRepoOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	var allIssues []*github.Issue
	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, i.Owner, fanoutRepo, opt)
		if err != nil {
			return "", fmt.Errorf("error listing orgs: %w", err)
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}
	for _, issue := range allIssues {
		if issue.GetTitle() == i.Title {
			return issue.GetHTMLURL(), nil
		}
	}
	issue, resp, err := client.Issues.Create(ctx, i.Owner, fanoutRepo, &github.IssueRequest{
		Title: &i.Title,
		Body:  &i.Body,
	})

	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusCreated {
		errorBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("error creating issue: %s", errorBody)
	}
	return issue.GetHTMLURL(), nil
}
