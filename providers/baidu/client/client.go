package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	modelBaidu "github.com/onetreegrove/panctl/providers/baidu/model"
	"golang.org/x/time/rate"
)

const (
	apiBaseURL       = "https://pan.baidu.com/rest/2.0"
	oauthTokenURL    = "https://openapi.baidu.com/oauth/2.0/token"
	defaultUploadURL = "https://d.pcs.baidu.com"
)

type Client struct {
	http    *resty.Client
	cred    modelBaidu.Credential
	limiter *rate.Limiter
}

func New(requestsPerSecond float64) *Client {
	c := &Client{
		http: resty.New().
			SetTimeout(30*time.Second).
			SetHeader("User-Agent", "pan.baidu.com"),
	}
	if requestsPerSecond > 0 {
		c.limiter = rate.NewLimiter(rate.Limit(requestsPerSecond), 1)
	}
	return c
}

func (c *Client) ImportCredential(cred modelBaidu.Credential) {
	c.cred = cred.WithDefaults()
}

func (c *Client) Credential() modelBaidu.Credential {
	return c.cred
}

func (c *Client) Wait(ctx context.Context) error {
	if c.limiter == nil {
		return nil
	}
	return c.limiter.Wait(ctx)
}

func (c *Client) RefreshToken(ctx context.Context) error {
	if err := c.Wait(ctx); err != nil {
		return err
	}
	if c.cred.RefreshToken == "" {
		return fmt.Errorf("missing baidu refresh token")
	}
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
		Description  string `json:"error_description"`
	}
	resp, err := c.http.R().
		SetContext(ctx).
		SetResult(&tokenResp).
		SetQueryParams(map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": c.cred.RefreshToken,
			"client_id":     c.cred.ClientID,
			"client_secret": c.cred.ClientSecret,
		}).
		Get(oauthTokenURL)
	if err != nil {
		return err
	}
	if tokenResp.Error != "" {
		return fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.Description)
	}
	if resp.StatusCode() >= 400 || tokenResp.AccessToken == "" {
		return fmt.Errorf("baidu token refresh failed: %s", resp.String())
	}
	c.cred.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		c.cred.RefreshToken = tokenResp.RefreshToken
	}
	return nil
}

func (c *Client) ensureAccessToken(ctx context.Context) error {
	if c.cred.AccessToken != "" {
		return nil
	}
	return c.RefreshToken(ctx)
}

func (c *Client) request(ctx context.Context, method, url string, query, form map[string]string, result any) ([]byte, error) {
	if err := c.ensureAccessToken(ctx); err != nil {
		return nil, err
	}
	var body []byte
	for attempt := 0; attempt < 2; attempt++ {
		if err := c.Wait(ctx); err != nil {
			return nil, err
		}
		req := c.http.R().SetContext(ctx).SetQueryParam("access_token", c.cred.AccessToken)
		if query != nil {
			req.SetQueryParams(query)
		}
		if form != nil {
			req.SetFormData(form)
		}
		if result != nil {
			req.SetResult(result)
		}
		resp, err := req.Execute(method, url)
		if err != nil {
			return nil, err
		}
		body = resp.Body()
		errno := jsonInt(body, "errno")
		if errno == 111 || errno == -6 {
			c.cred.AccessToken = ""
			if err := c.RefreshToken(ctx); err != nil {
				return nil, err
			}
			continue
		}
		if errno != 0 {
			return nil, fmt.Errorf("baidu request failed errno: %d body: %s", errno, string(body))
		}
		if resp.StatusCode() >= 400 {
			return nil, fmt.Errorf("baidu http status %d: %s", resp.StatusCode(), string(body))
		}
		return body, nil
	}
	return nil, fmt.Errorf("baidu request failed after token refresh: %s", string(body))
}

func jsonInt(body []byte, key string) int {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func (c *Client) get(ctx context.Context, pathname string, params map[string]string, result any) ([]byte, error) {
	return c.request(ctx, http.MethodGet, apiBaseURL+pathname, params, nil, result)
}

func (c *Client) postForm(ctx context.Context, pathname string, params, form map[string]string, result any) ([]byte, error) {
	return c.request(ctx, http.MethodPost, apiBaseURL+pathname, params, form, result)
}
