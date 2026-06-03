package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onetreegrove/panctl/providers/aliyun/model"
)

func TestRefreshTokenSelectsResourceDrive(t *testing.T) {
	var tokenCalled bool
	var driveCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/access_token":
			tokenCalled = true
			_, _ = w.Write([]byte(`{"access_token":"access_1","refresh_token":"refresh_2"}`))
		case "/adrive/v1.0/user/getDriveInfo":
			driveCalled = true
			_, _ = w.Write([]byte(`{"user_id":"user_1","default_drive_id":"default_1","resource_drive_id":"resource_1","backup_drive_id":"backup_1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cred := model.Credential{RefreshToken: "refresh_1", ClientID: "cid", ClientSecret: "secret", DriveType: model.DriveTypeResource}.WithDefaults()
	c := New(Options{APIBaseURL: srv.URL, OAuthTokenURL: srv.URL + "/oauth/access_token", RequestsPerSecond: 0})
	c.ImportCredential(cred)
	if err := c.RefreshToken(context.Background()); err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}
	if !tokenCalled || !driveCalled {
		t.Fatalf("expected token and drive endpoints to be called")
	}
	got := c.Credential()
	if got.AccessToken != "access_1" || got.RefreshToken != "refresh_2" || got.DriveID != "resource_1" || got.UserID != "user_1" {
		t.Fatalf("unexpected credential: %+v", got)
	}
}

func TestRequestRefreshesExpiredAccessTokenOnce(t *testing.T) {
	var protectedCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/access_token":
			_, _ = w.Write([]byte(`{"access_token":"access_2","refresh_token":"refresh_2"}`))
		case "/adrive/v1.0/user/getDriveInfo":
			_, _ = w.Write([]byte(`{"user_id":"user_1","resource_drive_id":"resource_1"}`))
		case "/protected":
			protectedCalls++
			if protectedCalls == 1 {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"code":"AccessTokenExpired","message":"expired"}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(Options{APIBaseURL: srv.URL, OAuthTokenURL: srv.URL + "/oauth/access_token"})
	c.ImportCredential(model.Credential{RefreshToken: "refresh_1", AccessToken: "access_1", DriveID: "resource_1", UserID: "user_1"}.WithDefaults())
	var out map[string]bool
	if _, err := c.Request(context.Background(), limiterOther, "/protected", http.MethodPost, nil, &out); err != nil {
		t.Fatalf("request returned error: %v", err)
	}
	if protectedCalls != 2 || !out["ok"] {
		t.Fatalf("expected retry success, calls=%d out=%+v", protectedCalls, out)
	}
}
