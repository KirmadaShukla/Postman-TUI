package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"my-new-go/internal/models"
)

var varPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

type Result struct {
	StatusCode   int
	Status       string
	Headers      http.Header
	Body         string
	Duration     time.Duration
	ResolvedURL  string
	Error        string
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

	url := Substitute(req.URL, vars)
	body := Substitute(req.Body, vars)
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	var bodyReader io.Reader
	if body != "" && method != http.MethodGet && method != http.MethodHead {
		bodyReader = bytes.NewBufferString(body)
	}

	httpReq, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return Result{Error: err.Error(), ResolvedURL: url}
	}

	for _, h := range req.Headers {
		if !h.Enabled || strings.TrimSpace(h.Key) == "" {
			continue
		}
		httpReq.Header.Set(Substitute(h.Key, vars), Substitute(h.Value, vars))
	}

	// Many APIs (e.g. DummyJSON) ignore JSON bodies without this header.
	if bodyReader != nil && httpReq.Header.Get("Content-Type") == "" && looksLikeJSON(body) {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: timeout}
	start := time.Now()
	resp, err := client.Do(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		return Result{Error: err.Error(), ResolvedURL: url, Duration: elapsed}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2MB cap
	if err != nil {
		return Result{
			StatusCode:  resp.StatusCode,
			Status:      resp.Status,
			Headers:     resp.Header,
			Duration:    elapsed,
			ResolvedURL: url,
			Error:       fmt.Sprintf("read body: %v", err),
		}
	}

	return Result{
		StatusCode:  resp.StatusCode,
		Status:      resp.Status,
		Headers:     resp.Header,
		Body:        string(raw),
		Duration:    elapsed,
		ResolvedURL: url,
	}
}

func looksLikeJSON(body string) bool {
	s := strings.TrimSpace(body)
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}
