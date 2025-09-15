package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	githubClient "github.com/google/go-github/v74/github"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const (
	stateKey    = "state"
	tokenKey    = "token"
	sessionName = "fan_out_work_github"
)

func NewOauthService(sessionStore *sessions.CookieStore) *OAuthService {
	return &OAuthService{
		oauthConfig:  githubOauthConfig,
		sessionStore: sessionStore,
		sessionName:  sessionName,
		verifier:     oauth2.GenerateVerifier(),
	}
}

type OAuthService struct {
	oauthConfig  *oauth2.Config
	sessionStore *sessions.CookieStore
	sessionName  string
	verifier     string
}

type OAuthCallbackParams struct {
	State string `query:"state"`
	Code  string `query:"code"`
}

func (os *OAuthService) RedirectURL(c echo.Context) string {
	state, err := generateRandomState()
	if err != nil {
		log.Fatal(err)
	}

	os.storeState(c, state)

	return os.oauthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(os.verifier))
}

func (os *OAuthService) StoreToken(c echo.Context) error {
	ctx := context.Background()
	fmt.Printf("Query params: %v", c.Request().URL.Query())
	var oauthCallbackParams OAuthCallbackParams
	err := c.Bind(&oauthCallbackParams)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Bound query params: %v", oauthCallbackParams)
	state, err := os.getState(c)
	if err != nil {
		return err
	}
	if state != oauthCallbackParams.State {
		log.Fatalf("state values doen't match: %v, %v", state, oauthCallbackParams.State)
	}
	token, err := os.oauthConfig.Exchange(ctx, oauthCallbackParams.Code, oauth2.VerifierOption(os.verifier))
	if err != nil {
		log.Fatal(err)
	}
	tokenJson, err := json.Marshal(token)
	if err != nil {
		log.Fatal(err)
	}
	os.storeToken(c, string(tokenJson))
	return nil
}

func (os *OAuthService) GetClient(c echo.Context) (*githubClient.Client, error) {
	token, err := os.getToken(c)
	if err != nil {
		return nil, err
	}
	return githubClient.NewClient(nil).WithAuthToken(token.AccessToken), nil
}

func (os *OAuthService) getState(c echo.Context) (string, error) {
	v, err := os.get(c, stateKey)
	if err != nil {
		return "", err
	}
	return v, nil
}

func (os *OAuthService) storeState(c echo.Context, value string) error {
	return os.store(c, stateKey, value)
}

func (os *OAuthService) getToken(c echo.Context) (oauth2.Token, error) {
	tokenJSON, err := os.get(c, tokenKey)
	if err != nil {
		return oauth2.Token{}, err
	}
	var token oauth2.Token
	err = json.Unmarshal([]byte(tokenJSON), &token)
	if err != nil {
		return oauth2.Token{}, err
	}
	return token, nil
}

func (os *OAuthService) storeToken(c echo.Context, value string) {
	os.store(c, tokenKey, value)
}

func (os *OAuthService) store(c echo.Context, key string, value string) error {
	session, _ := os.sessionStore.Get(c.Request(), os.sessionName)
	session.Values[key] = value
	return session.Save(c.Request(), c.Response())
}

func (os *OAuthService) get(c echo.Context, key string) (string, error) {
	session, _ := os.sessionStore.Get(c.Request(), os.sessionName)
	if v, ok := session.Values[key].(string); ok {
		return v, nil
	}
	return "", errors.New("key not found")
}

var githubOauthConfig = &oauth2.Config{
	ClientID:     os.Getenv("GITHUB_OAUTH_CLIENT_ID"),
	ClientSecret: os.Getenv("GITHUB_OAUTH_CLIENT_SECRET"),
	RedirectURL:  os.Getenv("GITHUB_OAUTH_REDIRECT_URL"),
	Scopes:       strings.Split(os.Getenv("GITHUB_OAUTH_SCOPES"), ","),
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
