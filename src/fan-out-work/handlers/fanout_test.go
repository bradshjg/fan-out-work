package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/bradshjg/fan-out-work/services"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type mockFanoutService struct{}

func (*mockFanoutService) ClearSession(c echo.Context) {
}

func (*mockFanoutService) AccessToken(c echo.Context) (string, error) {
	return "access-token", nil
}

func (*mockFanoutService) Orgs(c echo.Context) ([]string, error) {
	orgs := []string{"howdy", "there"}
	return orgs, nil
}

func (*mockFanoutService) Patches() ([]string, error) {
	return []string{"foo", "bar"}, nil
}

func (*mockFanoutService) Run(pr services.PatchRun) (string, error) {
	return "token", nil
}

func (*mockFanoutService) Status(pr services.PatchRun) ([]string, error) {
	return []string{"token"}, nil
}

func (*mockFanoutService) Output(token string) ([]string, bool, error) {
	return []string{}, true, nil
}

func TestHomeHandler(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := NewFanoutHandler(&mockFanoutService{})
	if assert.NoError(t, h.HomeHandler(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(rec.Body.String()))
		if err != nil {
			t.Fatalf("Failed to create goquery document: %v", err)
		}
		selection := doc.Find(`[data-testid="orgs"]`)
		if selection.Length() == 0 {
			t.Error("Element with data-testid='orgs' not found")
		}

		// Assert on the text content
		expectedText := "Select an org:  howdythere"
		if text := selection.Text(); text != expectedText {
			t.Errorf("Expected text '%s', got '%s'", expectedText, text)
		}
	}
}
