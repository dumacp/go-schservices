package nastclient

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

func TokenSource(ctx context.Context, username, password, url, realm, clientid, clientsecret string) (oauth2.TokenSource, error) {

	issuer := fmt.Sprintf("%s/realms/%s", url, realm)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	config := &oauth2.Config{
		ClientID:     clientid,
		ClientSecret: clientsecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  url,
		Scopes:       []string{oidc.ScopeOpenID},
	}

	tk, err := config.PasswordCredentialsToken(ctx, username, password)
	if err != nil {
		return nil, err
	}
	ts := config.TokenSource(ctx, tk)
	return ts, nil

}
