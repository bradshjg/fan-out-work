package services

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"sync"

	"github.com/stretchr/testify/assert/yaml"
)

var (
	outputMap = sync.Map{}
	patchDir  = "./patches"
)

type config struct {
	Branch  string `yaml:"branch"`
	PRTitle string `yaml:"pr-title"`
	PRBody  string `yaml:"pr-body"`
}

type PatchRun struct {
	accessToken string
	org         string
	patchName   string
	dryRun      bool
	executor    executor
}

type FanoutService interface {
	Patches() ([]string, error)
	Run(pr PatchRun) (string, error)
	Output(token string) (chan string, error)
}

func NewFanoutService() FanoutService {
	return &FanoutServiceImpl{}
}

type executorRun struct {
	args       []string
	streamName string
}

type executor interface {
	Run(er executorRun) error
}

// generateStreamName generates a cryptographically secure random string for output streams.
func generateStreamName() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

type executorImpl struct {
	streamName string
}

func (ex *executorImpl) Run(er executorRun) error {
	cmd := exec.Command("multi-gitter", er.args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error creating StdoutPipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting command: %v", err)
	}

	ch := make(chan string, 10)
	outputMap.Store(ex.streamName, ch)

	go func() {
		defer ex.closeOutputChannel(ch)
		scanner := bufio.NewScanner(stdoutPipe)

		for scanner.Scan() {
			line := scanner.Text()
			ch <- line
		}

		if err := scanner.Err(); err != nil {
			log.Printf("error reading stdout: %v", err)
		}

		if err := cmd.Wait(); err != nil {
			log.Printf("command finished with error: %v", err)
		}
	}()
	return nil
}

func (ex *executorImpl) closeOutputChannel(ch chan string) {
	close(ch)
	outputMap.Delete(ex.streamName)
}

type FanoutServiceImpl struct{}

func (*FanoutServiceImpl) Patches() ([]string, error) {
	var patches []string
	entries, err := os.ReadDir(patchDir)
	if err != nil {
		return []string{}, err
	}
	for _, e := range entries {
		if e.IsDir() {
			patches = append(patches, e.Name())
		}
	}
	return patches, nil
}

func (fs *FanoutServiceImpl) Run(pr PatchRun) (string, error) {
	possiblePatches, err := fs.Patches()
	if err != nil {
		return "", err
	}
	if !slices.Contains(possiblePatches, pr.patchName) {
		return "", fmt.Errorf("invalid patch name: %s", pr.patchName)
	}
	args, err := fs.execArgs(pr)
	if err != nil {
		return "", err
	}
	streamName, err := generateStreamName()
	if err != nil {
		return "", err
	}
	var executor executor
	if pr.executor == nil {
		executor = &executorImpl{}
	} else {
		executor = pr.executor
	}
	executorRun := executorRun{
		args:       args,
		streamName: streamName,
	}
	err = executor.Run(executorRun)
	if err != nil {
		return "", err
	}
	return executorRun.streamName, nil
}

func (*FanoutServiceImpl) Output(streamName string) (chan string, error) {
	ch, ok := outputMap.Load(streamName)
	if !ok {
		return nil, fmt.Errorf("no stream found for name %s", streamName)
	}
	return ch.(chan string), nil
}

func (fs *FanoutServiceImpl) execArgs(pr PatchRun) ([]string, error) {
	cfg, err := fs.patchConfig(pr)
	if err != nil {
		return []string{}, err
	}
	patch := fmt.Sprintf("patches/%s/patch", pr.patchName)
	args := []string{
		"run",
		patch,
		"--token", pr.accessToken,
		"--org", pr.org,
		"--branch", cfg.Branch,
		"--pr-title", cfg.PRTitle,
		"--pr-body", cfg.PRBody,
	}
	if pr.dryRun {
		args = append(args, "--log-level", "debug", "--dry-run")
	}

	return args, nil
}

func (fs *FanoutServiceImpl) patchConfig(pr PatchRun) (config, error) {
	patchesRoot, err := os.OpenRoot(patchDir)
	if err != nil {
		return config{}, err
	}
	patchRoot, err := patchesRoot.OpenRoot(pr.patchName)
	if err != nil {
		return config{}, err
	}
	cfgData, err := patchRoot.ReadFile("config.yml")
	if err != nil {
		return config{}, err
	}
	var cfg config
	err = yaml.Unmarshal(cfgData, &cfg)
	if err != nil {
		return config{}, err
	}
	return cfg, nil
}
