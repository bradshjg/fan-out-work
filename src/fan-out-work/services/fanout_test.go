package services

import (
	"os"
	"testing"

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

func TestRun(t *testing.T) {
	defer chdir(t, "..")()
	capturedArgs = []string{} // reset arg capture
	fs := NewFanoutService()
	pr := PatchRun{
		AccessToken: "gh-api-token",
		Org:         "gh-org",
		PatchName:   "example",
		DryRun:      false,
		Executor:    &mockExecutor{},
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
	fs := NewFanoutService()
	pr := PatchRun{
		AccessToken: "gh-api-token",
		Org:         "gh-org",
		PatchName:   "example",
		DryRun:      true,
		Executor:    &mockExecutor{},
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
	fs := NewFanoutService()
	pr := PatchRun{
		AccessToken: "gh-api-token",
		Org:         "gh-org",
		PatchName:   "../../invalid-patch",
		DryRun:      true,
		Executor:    &mockExecutor{},
	}
	_, err := fs.Run(pr)
	assert.NotNil(t, err, "Expected error got nil")
	expectedError := "invalid patch name: ../../invalid-patch"
	assert.Equal(t, expectedError, err.Error())
}
