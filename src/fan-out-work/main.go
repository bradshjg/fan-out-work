package main

import (
	"os"

	"github.com/bradshjg/fan-out-work/handlers"
	fanoutMiddleware "github.com/bradshjg/fan-out-work/middleware"
	"github.com/bradshjg/fan-out-work/services"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.Debug = os.Getenv("DEBUG") == "true"
	e.HideBanner = true
	e.HidePort = true
	e.Use(fanoutMiddleware.LoggingMiddleware())
	e.Use(fanoutMiddleware.RequestLoggingMiddleware())
	e.DisableHTTP2 = true
	e.Static("/static", "assets")
	e.HTTPErrorHandler = handlers.HTTPErrorHandler
	sessionStore := sessions.NewCookieStore(securecookie.GenerateRandomKey(32), securecookie.GenerateRandomKey(32))
	sessionStore.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 86400, // 1 day
	}
	e.Use(session.Middleware(sessionStore))

	os := services.NewOauthService(sessionStore)
	gs := services.NewGitHubService(os)
	fs := services.NewFanoutService(gs)

	fh := handlers.NewFanoutHandler(fs)
	gh := handlers.NewGitHubHandler(os)

	e.GET("/", fh.HomeHandler)
	e.POST("/run", fh.RunHandler)
	e.GET("/output", fh.OutputHandler)
	e.GET("/github/login", gh.OAuthHandler)
	e.GET("/github/callback", gh.OAuthCallbackHandler)
	e.GET("/*", handlers.RouteNotFoundHandler)

	e.Logger.Fatal(e.Start(":8080"))
}
