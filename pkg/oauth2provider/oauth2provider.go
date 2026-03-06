// Package oauth2provider provides pre-configured OAuth2 provider settings
// for common cloud storage services including Dropbox, Google Drive, and OneDrive.
package oauth2provider

import (
	"fmt"
	"net/url"
	"strings"
)

// Provider holds OAuth2 configuration for a specific service.
type Provider struct {
	Name             string
	AuthorizationURL string
	TokenURL         string
	Scopes           []string
}

// AuthURL generates an authorization URL with the given parameters.
func (p *Provider) AuthURL(clientID, redirectURI, state string) string {
	params := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
	}
	if len(p.Scopes) > 0 {
		params.Set("scope", strings.Join(p.Scopes, " "))
	}
	if state != "" {
		params.Set("state", state)
	}
	return fmt.Sprintf("%s?%s", p.AuthorizationURL, params.Encode())
}

// Dropbox returns a Provider configured for Dropbox OAuth2.
func Dropbox() *Provider {
	return &Provider{
		Name:             "Dropbox",
		AuthorizationURL: "https://www.dropbox.com/oauth2/authorize",
		TokenURL:         "https://api.dropboxapi.com/oauth2/token",
		Scopes:           []string{"files.content.write", "files.content.read", "files.metadata.read"},
	}
}

// GoogleDrive returns a Provider configured for Google Drive OAuth2.
func GoogleDrive() *Provider {
	return &Provider{
		Name:             "Google Drive",
		AuthorizationURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:         "https://oauth2.googleapis.com/token",
		Scopes: []string{
			"https://www.googleapis.com/auth/drive.file",
			"https://www.googleapis.com/auth/drive.readonly",
		},
	}
}

// OneDrive returns a Provider configured for Microsoft OneDrive OAuth2.
func OneDrive() *Provider {
	return &Provider{
		Name:             "OneDrive",
		AuthorizationURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL:         "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		Scopes:           []string{"Files.ReadWrite", "Files.ReadWrite.All", "offline_access"},
	}
}
