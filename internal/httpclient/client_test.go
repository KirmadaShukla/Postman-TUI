package httpclient

import "testing"

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
