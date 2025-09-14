package main

import (
	"bufio"
	"context"
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
	key           = []byte(os.Getenv("SESSION_SIGNING_KEY"))
	store         = sessions.NewCookieStore(key)
	verifier      = oauth2.GenerateVerifier()
	outputChannel = &OutputChannel{}
)

var githubOauthConfig = &oauth2.Config{
	ClientID:     os.Getenv("GITHUB_OAUTH_CLIENT_ID"),
	ClientSecret: os.Getenv("GITHUB_OAUTH_CLIENT_SECRET"),
	RedirectURL:  "http://localhost:8080/github/callback",
	Scopes:       []string{"user:email", "read:user"},
	Endpoint:     github.Endpoint,
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

func secret(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")

	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Print secret message
	fmt.Fprintln(w, "The cake is a lie!")
}

func login(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")

	// Authentication goes here
	// ...

	// Set user as authenticated
	session.Values["authenticated"] = true
	session.Save(r, w)
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(r, w)
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

func githubLoginHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r,
		githubOauthConfig.AuthCodeURL("state", oauth2.S256ChallengeOption(verifier)),
		http.StatusTemporaryRedirect,
	)
}

func githubCallback(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	code := r.FormValue("code")
	tok, err := githubOauthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		log.Fatal(err)
	}

	client := githubOauthConfig.Client(ctx, tok)
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
	http.HandleFunc("/secret", secret)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/run", run)
	http.HandleFunc("/output", output)
	http.HandleFunc("/github/login", githubLoginHandler)
	http.HandleFunc("/github/callback", githubCallback)

	fmt.Println("Server starting on port 8080...")
	http.ListenAndServe(":8080", nil)
}
