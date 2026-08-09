package httpclient

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"my-new-go/internal/models"
)

var varPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

type Result struct {
	StatusCode  int
	Status      string
	Headers     http.Header
	Body        string
	Duration    time.Duration
	ResolvedURL string
	Error       string
}

func Substitute(input string, vars map[string]string) string {
	if vars == nil {
		return input
	}
	return varPattern.ReplaceAllStringFunc(input, func(match string) string {
		key := varPattern.FindStringSubmatch(match)[1]
		if v, ok := vars[key]; ok {
			return v
		}
		return match
	})
}

func Send(req models.Request, vars map[string]string, timeout time.Duration) Result {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	rawURL := Substitute(req.URL, vars)
	rawURL = applyQueryParams(rawURL, req.Params, vars)
	rawURL = applyAuthQuery(rawURL, req.Auth, vars)

	body := Substitute(req.Body, vars)
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	var bodyReader io.Reader
	if body != "" && method != http.MethodGet && method != http.MethodHead {
		bodyReader = bytes.NewBufferString(body)
	}

	httpReq, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		return Result{Error: err.Error(), ResolvedURL: rawURL}
	}

	for _, h := range req.Headers {
		if !h.Enabled || strings.TrimSpace(h.Key) == "" {
			continue
		}
		httpReq.Header.Set(Substitute(h.Key, vars), Substitute(h.Value, vars))
	}
	applyAuthHeaders(httpReq, req.Auth, vars)

	// Many APIs (e.g. DummyJSON) ignore JSON bodies without this header.
	if bodyReader != nil && httpReq.Header.Get("Content-Type") == "" && looksLikeJSON(body) {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: timeout}
	start := time.Now()
	resp, err := client.Do(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		return Result{Error: err.Error(), ResolvedURL: rawURL, Duration: elapsed}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2MB cap
	if err != nil {
		return Result{
			StatusCode:  resp.StatusCode,
			Status:      resp.Status,
			Headers:     resp.Header,
			Duration:    elapsed,
			ResolvedURL: rawURL,
			Error:       fmt.Sprintf("read body: %v", err),
		}
	}

	return Result{
		StatusCode:  resp.StatusCode,
		Status:      resp.Status,
		Headers:     resp.Header,
		Body:        string(raw),
		Duration:    elapsed,
		ResolvedURL: rawURL,
	}
}

func applyQueryParams(rawURL string, params []models.Header, vars map[string]string) string {
	if len(params) == 0 {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	for _, p := range params {
		if !p.Enabled || strings.TrimSpace(p.Key) == "" {
			continue
		}
		q.Set(Substitute(p.Key, vars), Substitute(p.Value, vars))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func applyAuthQuery(rawURL string, auth models.Auth, vars map[string]string) string {
	if strings.ToLower(auth.Type) != models.AuthAPIKey {
		return rawURL
	}
	if strings.ToLower(auth.AddTo) != "query" {
		return rawURL
	}
	key := strings.TrimSpace(Substitute(auth.Key, vars))
	if key == "" {
		return rawURL
	}
	return applyQueryParams(rawURL, []models.Header{{
		Key: key, Value: Substitute(auth.Value, vars), Enabled: true,
	}}, nil)
}

func applyAuthHeaders(httpReq *http.Request, auth models.Auth, vars map[string]string) {
	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case models.AuthBearer:
		tok := strings.TrimSpace(Substitute(auth.Token, vars))
		if tok != "" {
			httpReq.Header.Set("Authorization", "Bearer "+tok)
		}
	case models.AuthBasic:
		user := Substitute(auth.Username, vars)
		pass := Substitute(auth.Password, vars)
		cred := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		httpReq.Header.Set("Authorization", "Basic "+cred)
	case models.AuthAPIKey:
		if strings.ToLower(auth.AddTo) == "query" {
			return
		}
		key := strings.TrimSpace(Substitute(auth.Key, vars))
		if key == "" {
			key = "X-API-Key"
		}
		httpReq.Header.Set(key, Substitute(auth.Value, vars))
	}
}

func looksLikeJSON(body string) bool {
	s := strings.TrimSpace(body)
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}
