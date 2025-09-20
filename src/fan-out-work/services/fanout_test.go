package services

import (
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type mockExecutor struct{}

var capturedArgs []string

func (*mockExecutor) Run(er executorRun) error {
	capturedArgs = er.args
	return nil
}

func chdir(t *testing.T, dir string) func() {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restoring working directory: %v", err)
		}
	}
}

func NewMockFanoutService() FanoutService {
	return &FanoutServiceImpl{
		githubService:    &mockGitHubService{},
		patchRunExecutor: &mockExecutor{},
	}
}

type mockGitHubService struct {
}

func (*mockGitHubService) ClearSession(c echo.Context) {
}

func (*mockGitHubService) AccessToken(c echo.Context) (string, error) {
	return "access-token", nil
}

func (*mockGitHubService) Orgs(c echo.Context) ([]string, error) {
	orgs := []string{"howdy", "there"}
	return orgs, nil
}

func TestRun(t *testing.T) {
	defer chdir(t, "..")()
	capturedArgs = []string{} // reset arg capture
	fs := NewMockFanoutService()
	pr := PatchRun{
		AccessToken: "gh-api-token",
		Org:         "gh-org",
		Patch:       "example",
		DryRun:      false,
	}
	_, err := fs.Run(pr)
	expectedArgs := []string{ // see patches/example/config.yml
		"run",
		"patches/example/patch",
		"--token", "gh-api-token",
		"--org", "gh-org",
		"--branch", "example-patch-pr-branch",
		"--pr-title", "Example PR Title",
		"--pr-body", "This is an example PR body.\n\nIt might contain multiple lines.\n",
		"--plain-output",
	}
	assert.Nil(t, err, "Expected nil error, got %v", err)
	assert.Equal(t, expectedArgs, capturedArgs, "Expected %v to be %v", capturedArgs, expectedArgs)
}

func TestDryRun(t *testing.T) {
	defer chdir(t, "..")()
	capturedArgs = []string{} // reset arg capture
	fs := NewMockFanoutService()
	pr := PatchRun{
		AccessToken: "gh-api-token",
		Org:         "gh-org",
		Patch:       "example",
		DryRun:      true,
	}
	_, err := fs.Run(pr)
	expectedArgs := []string{ // see patches/example/config.yml
		"run",
		"patches/example/patch",
		"--token", "gh-api-token",
		"--org", "gh-org",
		"--branch", "example-patch-pr-branch",
		"--pr-title", "Example PR Title",
		"--pr-body", "This is an example PR body.\n\nIt might contain multiple lines.\n",
		"--plain-output",
		"--log-level", "debug",
		"--dry-run",
	}
	assert.Nil(t, err, "Expected nil error, got %v", err)
	assert.Equal(t, expectedArgs, capturedArgs, "Expected %v to be %v", capturedArgs, expectedArgs)
}

func TestInvalidPatchName(t *testing.T) {
	defer chdir(t, "..")()
	capturedArgs = []string{} // reset arg capture
	fs := NewMockFanoutService()
	pr := PatchRun{
		AccessToken: "gh-api-token",
		Org:         "gh-org",
		Patch:       "../../invalid-patch",
		DryRun:      true,
	}
	_, err := fs.Run(pr)
	assert.NotNil(t, err, "Expected error got nil")
	expectedError := "invalid patch name: ../../invalid-patch"
	assert.Equal(t, expectedError, err.Error())
}
