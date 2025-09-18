package services

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
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
	AccessToken string
	Org         string
	PatchName   string
	DryRun      bool
	Executor    executor
}

type FanoutService interface {
	Patches() ([]string, error)
	Run(pr PatchRun) (string, error)
	Output(token string) ([]string, bool, error)
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
	name := base64.URLEncoding.EncodeToString(b)
	return name, nil
}

type executorImpl struct{}

func (ex *executorImpl) Run(er executorRun) error {
	cmd := exec.Command("multi-gitter", er.args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	ch := make(chan string, 10)
	outputMap.Store(er.streamName, ch)

	go func() {
		defer close(ch)
		var wg sync.WaitGroup

		wg.Go(func() {
			collectOutput(ch, stdoutPipe)
		})

		wg.Go(func() {
			collectOutput(ch, stderrPipe)
		})

		if err := cmd.Wait(); err != nil {
			log.Printf("command finished with error: %v", err)
		}

		wg.Wait()
	}()
	return nil
}

func collectOutput(ch chan string, readPipe io.ReadCloser) {
	scanner := bufio.NewScanner(readPipe)
	for scanner.Scan() {
		ch <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error reading pipe: %v", err)
	}
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
	if !slices.Contains(possiblePatches, pr.PatchName) {
		return "", fmt.Errorf("invalid patch name: %s", pr.PatchName)
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
	if pr.Executor == nil {
		executor = &executorImpl{}
	} else {
		executor = pr.Executor
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

func (*FanoutServiceImpl) Output(streamName string) ([]string, bool, error) {
	ch, ok := outputMap.Load(streamName)
	if !ok {
		return []string{}, false, fmt.Errorf("no stream found for name %s", streamName)
	}
	var outputLines []string
	for {
		select {
		case line, ok := <-ch.(chan string):
			if !ok {
				outputMap.Delete(streamName)
				return outputLines, true, nil
			}
			outputLines = append(outputLines, line)
		default:
			return outputLines, false, nil
		}
	}
}

func (fs *FanoutServiceImpl) execArgs(pr PatchRun) ([]string, error) {
	cfg, err := fs.patchConfig(pr)
	if err != nil {
		return []string{}, err
	}
	patch := fmt.Sprintf("patches/%s/patch", pr.PatchName)
	args := []string{
		"run",
		patch,
		"--token", pr.AccessToken,
		"--org", pr.Org,
		"--branch", cfg.Branch,
		"--pr-title", cfg.PRTitle,
		"--pr-body", cfg.PRBody,
		"--plain-output",
	}
	if pr.DryRun {
		args = append(args, "--log-level", "debug", "--dry-run")
	}

	return args, nil
}

func (fs *FanoutServiceImpl) patchConfig(pr PatchRun) (config, error) {
	patchesRoot, err := os.OpenRoot(patchDir)
	if err != nil {
		return config{}, err
	}
	patchRoot, err := patchesRoot.OpenRoot(pr.PatchName)
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
