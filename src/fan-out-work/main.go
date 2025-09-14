package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sync"

	"github.com/gorilla/sessions"
)

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key            = []byte("super-secret-key")
	store          = sessions.NewCookieStore(key)
	outputMap      = make(map[string]chan string)
	outputMapMutex sync.Mutex
)

func createOutputChannel(name string) chan string {
	ch := make(chan string, 5)
	outputMapMutex.Lock()
	defer outputMapMutex.Unlock()
	outputMap[name] = ch
	return ch
}

func getOutputChannel(name string) (chan string, error) {
	outputMapMutex.Lock()
	defer outputMapMutex.Unlock()
	ch, ok := outputMap[name]
	if !ok {
		return nil, fmt.Errorf("no output exists for key '%s'", name)
	}
	return ch, nil
}

func closeOutputChannel(name string) error {
	outputMapMutex.Lock()
	defer outputMapMutex.Unlock()
	ch, ok := outputMap[name]
	if !ok {
		return nil
	}
	close(ch)
	delete(outputMap, name)
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
	ch := createOutputChannel(outputChannelName)

	go func() {
		defer closeOutputChannel(outputChannelName)
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
	ch, err := getOutputChannel(outputChannelName)
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

func main() {
	http.HandleFunc("/secret", secret)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/run", run)
	http.HandleFunc("/output", output)

	fmt.Println("Server starting on port 8080...")
	http.ListenAndServe(":8080", nil)
}
