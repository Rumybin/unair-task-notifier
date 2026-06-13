package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	loginTimeout = 30 * time.Second
)

// Login melakukan autentikasi ke portal HEBAT Unair (Moodle).
// Mengembalikan http.CookieJar yang berisi session cookie (MoodleSession)
// jika login berhasil.
func Login(ctx context.Context, baseURL, username, password string) (http.CookieJar, error) {
	loginURL := baseURL + "/login/index.php"

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("auth: create cookie jar: %w", err)
	}

	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Step 1: GET login page, ambil logintoken
	token, err := fetchLoginToken(ctx, client, loginURL)
	if err != nil {
		return nil, fmt.Errorf("auth: fetch login token: %w", err)
	}

	// Step 2: POST credentials
	if err := postLogin(ctx, client, loginURL, username, password, token); err != nil {
		return nil, fmt.Errorf("auth: post login: %w", err)
	}

	// Verifikasi: MoodleSession cookie harus ada
	u, _ := url.Parse(baseURL)
	cookies := jar.Cookies(u)
	for _, c := range cookies {
		if c.Name == "MoodleSession" {
			return jar, nil
		}
	}

	return nil, fmt.Errorf("auth: MoodleSession cookie not found after login")
}

func fetchLoginToken(ctx context.Context, client *http.Client, loginURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, loginURL, nil)
	if err != nil {
		return "", fmt.Errorf("create GET request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", loginURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s returned status %d", loginURL, resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, 512*1024) // max 512KB
	return extractLoginToken(limited)
}

func extractLoginToken(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	var token string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if token != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "input" {
			var name, value string
			for _, attr := range n.Attr {
				if attr.Key == "name" {
					name = attr.Val
				}
				if attr.Key == "value" {
					value = attr.Val
				}
			}
			if name == "logintoken" {
				token = value
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if token == "" {
		return "", fmt.Errorf("logintoken not found in HTML")
	}
	return token, nil
}

func postLogin(ctx context.Context, client *http.Client, loginURL, username, password, token string) error {
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	form.Set("logintoken", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", loginURL, err)
	}
	defer resp.Body.Close()

	// Moodle merespon dengan redirect 303 ke /my/ jika sukses
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(body))
	}

	location := resp.Header.Get("Location")
	if !strings.Contains(location, "/my/") {
		return fmt.Errorf("unexpected redirect target: %s", location)
	}

	return nil
}


// NewHTTPClient membuat http.Client dengan CookieJar yang sudah berisi session.
// Gunakan client ini untuk request selanjutnya (scraping, dll).
func NewHTTPClient(jar http.CookieJar) *http.Client {
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("auth: too many redirects")
			}
			return nil
		},
	}
}
