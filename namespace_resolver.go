package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// looksLikeUUID reports whether s is in canonical UUID form (8-4-4-4-12 hex
// with hyphens). Used to skip namespace resolution when the user already
// supplied a raw namespace ID.
func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHex {
			return false
		}
	}
	return true
}

// NamespaceResolver maps a Nacos namespace display name to its UUID by calling
// the Nacos console HTTP API. It is used only at startup.
type NamespaceResolver struct {
	baseURL  string // e.g. http://host:port/nacos
	username string
	password string
	client   *http.Client
}

func newNamespaceResolver(baseURL, username, password string, timeout time.Duration) *NamespaceResolver {
	return &NamespaceResolver{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		client:   &http.Client{Timeout: timeout},
	}
}

type loginResponse struct {
	AccessToken string `json:"accessToken"`
}

// login authenticates against /v1/auth/login and returns an accessToken.
func (r *NamespaceResolver) login() (string, error) {
	resp, err := r.client.PostForm(r.baseURL+"/v1/auth/login", url.Values{
		"username": {r.username},
		"password": {r.password},
	})
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read login response failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var lr loginResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return "", fmt.Errorf("parse login response failed: %w", err)
	}
	if lr.AccessToken == "" {
		return "", fmt.Errorf("login response missing accessToken")
	}
	return lr.AccessToken, nil
}

// Namespace is one entry from the Nacos console namespace list.
type Namespace struct {
	Namespace         string `json:"namespace"`         // UUID ("" for public)
	NamespaceShowName string `json:"namespaceShowName"` // human-friendly display name
	ConfigCount       int    `json:"configCount"`
	Type              int    `json:"type"`
}

type namespacesResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    []Namespace `json:"data"`
}

// fetchNamespaces retrieves the full namespace list from the console API.
func (r *NamespaceResolver) fetchNamespaces(token string) ([]Namespace, error) {
	reqURL := fmt.Sprintf("%s/v1/console/namespaces?accessToken=%s", r.baseURL, url.QueryEscape(token))
	resp, err := r.client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("fetch namespaces request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read namespaces response failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch namespaces failed with status %d: %s", resp.StatusCode, string(body))
	}

	var nr namespacesResponse
	if err := json.Unmarshal(body, &nr); err != nil {
		return nil, fmt.Errorf("parse namespaces response failed: %w", err)
	}
	return nr.Data, nil
}

// Resolve logs in, fetches the namespace list, and returns the UUID for the
// given input. It matches on display name first, then on UUID. On no match it
// returns an error listing the available display names. Any HTTP/auth failure
// is returned as an error so the caller can fall back to the original value.
func (r *NamespaceResolver) Resolve(input string) (string, bool, error) {
	token, err := r.login()
	if err != nil {
		return "", false, err
	}
	namespaces, err := r.fetchNamespaces(token)
	if err != nil {
		return "", false, err
	}

	for _, ns := range namespaces {
		if ns.NamespaceShowName == input {
			return ns.Namespace, true, nil
		}
	}
	for _, ns := range namespaces {
		if ns.Namespace == input {
			return ns.Namespace, true, nil
		}
	}

	available := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		available = append(available, ns.NamespaceShowName)
	}
	return "", false, fmt.Errorf("namespace %q not found, available: [%s]", input, strings.Join(available, ", "))
}
