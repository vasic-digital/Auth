package oauth2provider

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDropboxProvider(t *testing.T) {
	p := Dropbox()
	assert.Equal(t, "Dropbox", p.Name)
	assert.Contains(t, p.AuthorizationURL, "dropbox.com")
	assert.Contains(t, p.TokenURL, "dropboxapi.com")
	assert.NotEmpty(t, p.Scopes)
}

func TestGoogleDriveProvider(t *testing.T) {
	p := GoogleDrive()
	assert.Equal(t, "Google Drive", p.Name)
	assert.Contains(t, p.AuthorizationURL, "accounts.google.com")
	assert.Contains(t, p.TokenURL, "googleapis.com")
	assert.NotEmpty(t, p.Scopes)
}

func TestOneDriveProvider(t *testing.T) {
	p := OneDrive()
	assert.Equal(t, "OneDrive", p.Name)
	assert.Contains(t, p.AuthorizationURL, "microsoftonline.com")
	assert.Contains(t, p.TokenURL, "microsoftonline.com")
	assert.NotEmpty(t, p.Scopes)
}

func TestAuthURL_ContainsClientID(t *testing.T) {
	p := Dropbox()
	url := p.AuthURL("my-client-id", "https://app/callback", "")
	assert.Contains(t, url, "client_id=my-client-id")
}

func TestAuthURL_ContainsRedirectURI(t *testing.T) {
	p := Dropbox()
	url := p.AuthURL("cid", "https://app/callback", "")
	assert.Contains(t, url, "redirect_uri=")
}

func TestAuthURL_ContainsResponseTypeCode(t *testing.T) {
	p := Dropbox()
	url := p.AuthURL("cid", "https://app/callback", "")
	assert.Contains(t, url, "response_type=code")
}

func TestAuthURL_ContainsAccessTypeOffline(t *testing.T) {
	p := Dropbox()
	url := p.AuthURL("cid", "https://app/callback", "")
	assert.Contains(t, url, "access_type=offline")
}

func TestAuthURL_ContainsPromptConsent(t *testing.T) {
	p := Dropbox()
	url := p.AuthURL("cid", "https://app/callback", "")
	assert.Contains(t, url, "prompt=consent")
}

func TestAuthURL_ContainsScopes(t *testing.T) {
	p := Dropbox()
	url := p.AuthURL("cid", "https://app/callback", "")
	assert.Contains(t, url, "scope=")
	for _, scope := range p.Scopes {
		assert.True(t, strings.Contains(url, scope) || strings.Contains(url, strings.ReplaceAll(scope, ".", ".")),
			"URL should contain scope %s", scope)
	}
}

func TestAuthURL_ContainsState(t *testing.T) {
	p := Dropbox()
	url := p.AuthURL("cid", "https://app/callback", "my-csrf-token")
	assert.Contains(t, url, "state=my-csrf-token")
}

func TestAuthURL_OmitsStateWhenEmpty(t *testing.T) {
	p := Dropbox()
	url := p.AuthURL("cid", "https://app/callback", "")
	assert.NotContains(t, url, "state=")
}

func TestAuthURL_StartsWithAuthorizationURL(t *testing.T) {
	p := GoogleDrive()
	url := p.AuthURL("cid", "https://app/callback", "")
	assert.True(t, strings.HasPrefix(url, p.AuthorizationURL))
}

func TestAuthURL_NoScopes(t *testing.T) {
	p := &Provider{
		Name:             "Custom",
		AuthorizationURL: "https://custom.com/auth",
		TokenURL:         "https://custom.com/token",
		Scopes:           nil,
	}
	url := p.AuthURL("cid", "https://app/callback", "")
	assert.NotContains(t, url, "scope=")
}
