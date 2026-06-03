package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	modelAliyun "github.com/onetreegrove/panctl/providers/aliyun/model"
)

const (
	defaultAPIBaseURL    = "https://openapi.alipan.com"
	defaultOAuthTokenURL = "https://openapi.alipan.com/oauth/access_token"
)

type Options struct {
	APIBaseURL        string
	OAuthTokenURL     string
	RequestsPerSecond float64
}

type Client struct {
	http          *resty.Client
	apiBaseURL    string
	oauthTokenURL string
	cred          modelAliyun.Credential
	limiter       *userLimiter
}

type errResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e errResp) isZero() bool {
	return e.Code == "" && e.Message == ""
}

func (e errResp) Error() string {
	if e.Code == "" {
		return e.Message
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

func New(opts Options) *Client {
	apiBaseURL := opts.APIBaseURL
	if apiBaseURL == "" {
		apiBaseURL = defaultAPIBaseURL
	}
	oauthURL := opts.OAuthTokenURL
	if oauthURL == "" {
		oauthURL = defaultOAuthTokenURL
	}
	c := &Client{
		http: resty.New().
			SetTimeout(30*time.Second).
			SetHeader("User-Agent", "panctl").
			SetHeader("Content-Type", "application/json"),
		apiBaseURL:    strings.TrimRight(apiBaseURL, "/"),
		oauthTokenURL: oauthURL,
		limiter:       limiterForUser(globalLimiterKey),
	}
	if opts.RequestsPerSecond > 0 {
		c.limiter = newUserLimiter(opts.RequestsPerSecond)
	}
	return c
}

func (c *Client) ImportCredential(cred modelAliyun.Credential) {
	c.cred = cred.WithDefaults()
	if c.cred.UserID != "" {
		c.limiter = limiterForUser(c.cred.UserID)
	}
}

func (c *Client) Credential() modelAliyun.Credential {
	return c.cred
}

func (c *Client) RefreshToken(ctx context.Context) error {
	if c.cred.RefreshToken == "" {
		return fmt.Errorf("missing aliyun refresh token")
	}
	if err := c.limiter.wait(ctx, limiterOther); err != nil {
		return err
	}
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Code         string `json:"code"`
		Message      string `json:"message"`
		Text         string `json:"text"`
	}
	req := c.http.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult(&tokenResp).
		SetBody(map[string]string{
			"client_id":     c.cred.ClientID,
			"client_secret": c.cred.ClientSecret,
			"grant_type":    "refresh_token",
			"refresh_token": c.cred.RefreshToken,
		})
	resp, err := req.Post(c.oauthTokenURL)
	if err != nil {
		return err
	}
	if tokenResp.Code != "" || tokenResp.Text != "" || resp.StatusCode() >= 400 {
		msg := tokenResp.Message
		if msg == "" {
			msg = tokenResp.Text
		}
		if msg == "" {
			msg = resp.String()
		}
		return fmt.Errorf("failed to refresh aliyun token: %s", msg)
	}
	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		return fmt.Errorf("failed to refresh aliyun token: empty token response")
	}
	c.cred.AccessToken = tokenResp.AccessToken
	c.cred.RefreshToken = tokenResp.RefreshToken
	if err := c.loadDriveInfo(ctx); err != nil {
		return err
	}
	return nil
}

func (c *Client) loadDriveInfo(ctx context.Context) error {
	var driveResp struct {
		UserID          string `json:"user_id"`
		DefaultDriveID  string `json:"default_drive_id"`
		ResourceDriveID string `json:"resource_drive_id"`
		BackupDriveID   string `json:"backup_drive_id"`
	}
	if _, err := c.Request(ctx, limiterOther, "/adrive/v1.0/user/getDriveInfo", http.MethodPost, nil, &driveResp); err != nil {
		return err
	}
	c.cred.UserID = driveResp.UserID
	switch c.cred.DriveType {
	case modelAliyun.DriveTypeDefault:
		c.cred.DriveID = driveResp.DefaultDriveID
	case modelAliyun.DriveTypeBackup:
		c.cred.DriveID = driveResp.BackupDriveID
	default:
		c.cred.DriveID = driveResp.ResourceDriveID
	}
	if c.cred.DriveID == "" {
		c.cred.DriveID = driveResp.DefaultDriveID
	}
	if c.cred.UserID != "" {
		c.limiter = limiterForUser(c.cred.UserID)
	}
	return nil
}

func (c *Client) ensureAccessToken(ctx context.Context) error {
	if c.cred.AccessToken != "" {
		return nil
	}
	return c.RefreshToken(ctx)
}

func (c *Client) Request(ctx context.Context, typ limiterType, uri, method string, body any, result any) ([]byte, error) {
	return c.request(ctx, typ, uri, method, body, result, false)
}

func (c *Client) request(ctx context.Context, typ limiterType, uri, method string, body any, result any, retried bool) ([]byte, error) {
	if err := c.ensureAccessToken(ctx); err != nil {
		return nil, err
	}
	if err := c.limiter.wait(ctx, typ); err != nil {
		return nil, err
	}
	req := c.http.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetHeader("Authorization", "Bearer "+c.cred.AccessToken)
	if body != nil {
		req.SetBody(body)
	}
	resp, err := req.Execute(method, c.apiBaseURL+uri)
	if err != nil {
		return nil, err
	}
	bodyBytes := resp.Body()
	apiErr := parseErrResp(bodyBytes)
	if !apiErr.isZero() {
		if isAuthExpired(apiErr.Code) && !retried {
			c.cred.AccessToken = ""
			if err := c.RefreshToken(ctx); err != nil {
				return nil, err
			}
			return c.request(ctx, typ, uri, method, body, result, true)
		}
		return nil, apiErr
	}
	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("aliyun http status %d: %s", resp.StatusCode(), string(bodyBytes))
	}
	if result != nil {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return nil, err
		}
	}
	return bodyBytes, nil
}

func parseErrResp(body []byte) errResp {
	var e errResp
	if err := json.Unmarshal(body, &e); err != nil {
		return errResp{}
	}
	return e
}

func isAuthExpired(code string) bool {
	switch code {
	case "AccessTokenInvalid", "AccessTokenExpired", "I400JD":
		return true
	default:
		return false
	}
}
