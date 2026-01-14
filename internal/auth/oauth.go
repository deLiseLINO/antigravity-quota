package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/deLiseLINO/antigravity-quota/internal/config"
)

const (
	googleTokenURL = "https://oauth2.googleapis.com/token"
	googleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"

	// Antigravity OAuth credentials (shared across CLIProxyAPI ecosystem - public in open-source code)
	antigravityClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	antigravityClientSecret = "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"
)

func LoginFlow() (*config.TokenFile, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start local server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d", port)

	params := url.Values{}
	params.Set("client_id", antigravityClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	params.Set("scope", strings.Join([]string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
		"https://www.googleapis.com/auth/cclog",
		"https://www.googleapis.com/auth/experimentsandconfigs",
		"openid",
	}, " "))

	authURL := googleAuthURL + "?" + params.Encode()

	openBrowser(authURL)

	codeCh := make(chan string)
	errCh := make(chan error)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				return
			}
			code := r.URL.Query().Get("code")
			if code != "" {
				w.Write([]byte("Authentication successful! You can close this window now."))
				codeCh <- code
			} else {
				w.Write([]byte("Authentication failed! No code received."))
				errCh <- fmt.Errorf("no code received")
			}
		}),
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timed out")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	return exchangeCodeForToken(code, redirectURI)
}

func exchangeCodeForToken(code, redirectURI string) (*config.TokenFile, error) {
	data := url.Values{}
	data.Set("client_id", antigravityClientID)
	data.Set("client_secret", antigravityClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", googleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	type TokenExchangeResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}

	var tokenResp TokenExchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	newExpiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	tokenFile := &config.TokenFile{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		Expired:      newExpiry.Format(time.RFC3339),
		Type:         tokenResp.TokenType,
	}

	FetchUserEmail(tokenFile)

	configDir, err := config.EnsureConfigDir()
	if err != nil {
		return nil, err
	}

	tokenFile.FilePath = filepath.Join(configDir, fmt.Sprintf("antigravity_token_%s.json", config.SanitizeEmail(tokenFile.Email)))
	if err := config.SaveToken(tokenFile); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return tokenFile, nil
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	_ = err
}
