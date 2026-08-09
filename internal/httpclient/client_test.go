package httpclient

import (
	"strings"
	"testing"

	"my-new-go/internal/models"
)

func TestSubstitute(t *testing.T) {
	vars := map[string]string{
		"base_url": "https://api.example.com",
		"id":       "42",
	}
	got := Substitute("{{base_url}}/users/{{id}}", vars)
	want := "https://api.example.com/users/42"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if Substitute("{{missing}}", vars) != "{{missing}}" {
		t.Fatal("missing vars should stay intact")
	}
}

func TestLooksLikeJSON(t *testing.T) {
	if !looksLikeJSON(`{"userId":5}`) {
		t.Fatal("expected object to look like JSON")
	}
	if looksLikeJSON("plain text") {
		t.Fatal("plain text should not look like JSON")
	}
}

func TestApplyQueryParams(t *testing.T) {
	got := applyQueryParams("https://example.com/search?q=old", []models.Header{
		{Key: "q", Value: "new", Enabled: true},
		{Key: "page", Value: "2", Enabled: true},
		{Key: "skip", Value: "x", Enabled: false},
	}, nil)
	if !strings.Contains(got, "q=new") || !strings.Contains(got, "page=2") || strings.Contains(got, "skip=") {
		t.Fatalf("unexpected url: %s", got)
	}
}
