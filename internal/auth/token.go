package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/deLiseLINO/antigravity-quota/internal/config"
)

func IsExpired(expiredStr string) bool {
	if expiredStr == "" {
		return false
	}
	expiredTime, err := time.Parse(time.RFC3339, expiredStr)
	if err != nil {
		return false
	}
	return time.Now().After(expiredTime.Add(-5 * time.Minute))
}

func RefreshToken(tf *config.TokenFile) error {
	if tf.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("client_id", antigravityClientID)
	data.Set("client_secret", antigravityClientSecret)
	data.Set("refresh_token", tf.RefreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", googleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp config.TokenRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode refresh response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("received empty access token")
	}

	tf.AccessToken = tokenResp.AccessToken
	tf.ExpiresIn = tokenResp.ExpiresIn
	newExpiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	tf.Expired = newExpiry.Format(time.RFC3339)

	return config.SaveToken(tf)
}

func FetchUserEmail(tf *config.TokenFile) error {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tf.AccessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch user info: %d", resp.StatusCode)
	}

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return err
	}

	tf.Email = userInfo.Email
	return nil
}
