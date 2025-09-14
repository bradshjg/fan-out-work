package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/gorilla/sessions"
)

type OutputChannel struct {
	sync.Map
}

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key               = []byte(os.Getenv("SESSION_SIGNING_KEY"))
	store             = sessions.NewCookieStore(key)
	verifier          = oauth2.GenerateVerifier()
	outputChannel     = &OutputChannel{}
	sessionCookieName = "fanout-out-work-session"
)

var githubOauthConfig = &oauth2.Config{
	ClientID:     os.Getenv("GITHUB_OAUTH_CLIENT_ID"),
	ClientSecret: os.Getenv("GITHUB_OAUTH_CLIENT_SECRET"),
	RedirectURL:  "http://localhost:8080/github/callback",
	Scopes:       []string{"user:email", "read:user"},
	Endpoint:     github.Endpoint,
}

// generateRandomState generates a cryptographically secure random string for OAuth state.
func generateRandomState() (string, error) {
	b := make([]byte, 32) // Generate a 32-byte random string
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (output *OutputChannel) create(name string) chan string {
	ch := make(chan string, 5)
	output.Store(name, ch)
	return ch
}

func (output *OutputChannel) get(name string) (chan string, error) {
	ch, ok := output.Load(name)
	if !ok {
		return nil, fmt.Errorf("no output exists for key '%s'", name)
	}
	return ch.(chan string), nil
}

func (output *OutputChannel) close(name string) error {
	ch, err := output.get(name)
	if err != nil {
		return nil
	}
	close(ch)
	output.Delete(name)
	return nil
}

func run(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("./print.sh")

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error creating StdoutPipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting command: %v", err)
	}

	outputChannelName := r.URL.Query().Get("name")
	ch := outputChannel.create(outputChannelName)

	go func() {
		defer outputChannel.close(outputChannelName)
		scanner := bufio.NewScanner(stdoutPipe)

		for scanner.Scan() {
			line := scanner.Text()
			ch <- line
		}

		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading stdout: %v", err)
		}

		if err := cmd.Wait(); err != nil {
			log.Fatalf("Command finished with error: %v", err)
		}
	}()

	fmt.Fprintln(w, "Command running!")
}

func output(w http.ResponseWriter, r *http.Request) {
	outputChannelName := r.URL.Query().Get("name")
	ch, err := outputChannel.get(outputChannelName)
	if err != nil {
		fmt.Fprintf(w, "Error getting outout: %v", err)
		return
	}

	for {
		select {
		case msg := <-ch:
			fmt.Fprintln(w, msg)
		default:
			fmt.Fprintln(w, "No more messages, channel is empty.")
			return
		}
	}
}

func githubLogin(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionCookieName)
	state, err := generateRandomState()
	if err != nil {
		fmt.Fprintf(w, "Error generating state: %v", err)
		return
	}
	session.Values["state"] = state
	session.Save(r, w)

	http.Redirect(w, r,
		githubOauthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)),
		http.StatusTemporaryRedirect,
	)
}

func githubCallback(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionCookieName)
	ctx := context.Background()
	state := r.FormValue("state")
	code := r.FormValue("code")
	session_state := session.Values["state"]
	if session_state != state {
		fmt.Fprintf(w, "State values %v and %v didn't match!", session_state, state)
		return
	}
	token, err := githubOauthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		log.Fatal(err)
	}
	tokenJson, err := json.Marshal(token)
	if err != nil {
		log.Fatal(err)
	}
	session.Values["token"] = string(tokenJson)
	session.Save(r, w)
	http.Redirect(w, r,
		"/me",
		http.StatusTemporaryRedirect,
	)
}

func me(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionCookieName)
	ctx := context.Background()
	tokenJSON := session.Values["token"].(string)
	var token oauth2.Token
	json.Unmarshal([]byte(tokenJSON), &token)

	client := githubOauthConfig.Client(ctx, &token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	bodyString := string(bodyBytes)
	fmt.Fprintln(w, bodyString)

}

func main() {
	http.HandleFunc("/run", run)
	http.HandleFunc("/output", output)
	http.HandleFunc("/github/login", githubLogin)
	http.HandleFunc("/github/callback", githubCallback)
	http.HandleFunc("/me", me)

	fmt.Println("Server starting on port 8080...")
	http.ListenAndServe(":8080", nil)
}
