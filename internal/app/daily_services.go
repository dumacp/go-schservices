package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/dumacp/go-schservices/internal/utils"
	"golang.org/x/oauth2"
)

const (
	dailySvcURLFmt = "%s/api/external-system-gateway/rest/driver-daily-services"

	kcDailyRealm        = "DEVICES"
	kcDailyClientID     = "devices2"
	kcDailyClientSecret = "b73479a3-225b-4b96-ad65-22edd82623a3"
	kcDailyBaseURL      = "https://fleet.nebulae.com.co/auth"
	apiDailyBaseURL     = "https://fleet.nebulae.com.co"
)

// fetchDriverDailyServices calls the fleet API and returns raw JSON bytes.
func fetchDriverDailyServices(deviceID, driverDoc string) ([]byte, error) {
	c := &http.Client{
		Transport: utils.LoadLocalCert(),
		Timeout:   60 * time.Second,
	}
	ctx := context.WithValue(context.TODO(), oauth2.HTTPClient, c)

	issuer := fmt.Sprintf("%s/realms/%s", kcDailyBaseURL, kcDailyRealm)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc provider: %w", err)
	}

	config := &oauth2.Config{
		ClientID:     kcDailyClientID,
		ClientSecret: kcDailyClientSecret,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID},
	}

	tk, err := config.PasswordCredentialsToken(ctx, deviceID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("password grant: %w", err)
	}
	client := config.Client(ctx, tk)

	url := fmt.Sprintf("%s/%s?page=0&count=10", fmt.Sprintf(dailySvcURLFmt, apiDailyBaseURL), driverDoc)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
