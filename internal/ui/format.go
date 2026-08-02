package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"my-new-go/internal/httpclient"
)

// Response/body formatting helpers used by the view layer.

func methodStyle(method string) lipgloss.Style {
	if s, ok := methodStyles[strings.ToUpper(method)]; ok {
		return s
	}
	return valueStyle
}

func formatResponseTab(r httpclient.Result, tab int) string {
	if r.Error != "" && r.Body == "" && tab == 0 {
		return "Error: " + r.Error
	}
	if tab == 1 {
		if len(r.Headers) == 0 {
			if r.Status == "" {
				return "Send a request to see headers."
			}
			return "(no headers)"
		}
		keys := make([]string, 0, len(r.Headers))
		for k := range r.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("%s: %s\n", k, strings.Join(r.Headers[k], ", ")))
		}
		return b.String()
	}

	if r.Error != "" {
		return "Error: " + r.Error + "\n\n" + prettyBody(r.Body)
	}
	if r.Status == "" {
		return "Send a request to see the response body."
	}
	return prettyBody(r.Body)
}

func prettyBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "(empty)"
	}
	var anyJSON any
	if json.Unmarshal([]byte(body), &anyJSON) == nil {
		pretty, err := json.MarshalIndent(anyJSON, "", "  ")
		if err == nil {
			return string(pretty)
		}
	}
	return body
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
