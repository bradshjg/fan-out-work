package main

import (
	"github.com/bradshjg/fan-out-work/handlers"
	"github.com/bradshjg/fan-out-work/services"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

var (
	SessionAuthKey = securecookie.GenerateRandomKey(32)
	SessionEncKey  = securecookie.GenerateRandomKey(32)
)

func main() {
	e := echo.New()

	e.Static("/static", "assets")
	sessionStore := sessions.NewCookieStore(SessionAuthKey, SessionEncKey)
	e.Use(session.Middleware(sessionStore))
	os := services.NewOauthService(sessionStore)

	fh := handlers.NewFanoutHandler(services.NewGitHubService(os), services.NewFanoutService())
	gh := handlers.NewGitHubHandler(*os)

	e.GET("/", fh.HomeHandler)
	e.POST("/run", fh.RunHandler)
	e.GET("/output", fh.OutputHandler)
	e.GET("/github/login", gh.OAuthHandler)
	e.GET("/github/callback", gh.OAuthCallbackHandler)
	e.GET("/*", handlers.RouteNotFoundHandler)

	e.Logger.Fatal(e.Start(":8080"))
}
