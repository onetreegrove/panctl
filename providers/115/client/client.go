package client

import (
	"context"

	driver115 "github.com/SheltonZhu/115driver/pkg/driver"
	model115 "github.com/onetreegrove/panctl/providers/115/model"
	"golang.org/x/time/rate"
)

type Client struct {
	raw     *driver115.Pan115Client
	limiter *rate.Limiter
}

func New(requestsPerSecond float64) *Client {
	c := &Client{raw: driver115.New(driver115.UA("Mozilla/5.0 115Browser/" + DefaultAppVersion))}
	if requestsPerSecond > 0 {
		c.limiter = rate.NewLimiter(rate.Limit(requestsPerSecond), 1)
	}
	return c
}

func (c *Client) Wait(ctx context.Context) error {
	if c.limiter == nil {
		return nil
	}
	return c.limiter.Wait(ctx)
}

func (c *Client) init(ctx context.Context) error {
	if err := c.Wait(ctx); err != nil {
		return err
	}
	c.ensureAppVersion(ctx)
	return nil
}

func (c *Client) LoginCookie(ctx context.Context, cred model115.Credential) error {
	if err := c.init(ctx); err != nil {
		return err
	}
	rawCred := &driver115.Credential{}
	if err := rawCred.FromCookie(cred.Cookie()); err != nil {
		return err
	}
	c.raw.ImportCredential(rawCred)
	return c.raw.LoginCheck()
}

func (c *Client) QRCodeStart(ctx context.Context) (*driver115.QRCodeSession, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}
	return c.raw.QRCodeStart()
}

func (c *Client) QRCodeStatus(ctx context.Context, s *driver115.QRCodeSession) (*driver115.QRCodeStatus, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}
	return c.raw.QRCodeStatus(s)
}

func (c *Client) QRCodeLoginWithApp(ctx context.Context, s *driver115.QRCodeSession, source string) (*driver115.Credential, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}
	return c.raw.QRCodeLoginWithApp(s, driver115.LoginApp(source))
}
