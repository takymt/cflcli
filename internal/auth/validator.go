package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Validator interface {
	Validate(context.Context, Credentials) error
}

type HTTPValidator struct {
	client   *http.Client
	buildURL func(string) string
}

func NewHTTPValidator(client *http.Client) *HTTPValidator {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &HTTPValidator{
		client:   client,
		buildURL: defaultValidationURL,
	}
}

func (v *HTTPValidator) Validate(ctx context.Context, creds Credentials) error {
	required, err := RequireCredentials(creds)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.buildURL(required.Domain), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(required.Email, required.APIToken)

	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}

	raw, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		if closeErr != nil {
			return fmt.Errorf("validation response read failed: %v: %w", closeErr, readErr)
		}
		return fmt.Errorf("validation response read failed: %w", readErr)
	}
	if closeErr != nil {
		return closeErr
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body := strings.TrimSpace(string(raw))
	if body == "" {
		return fmt.Errorf("%s", resp.Status)
	}
	return fmt.Errorf("%s: %s", resp.Status, body)
}

func defaultValidationURL(domain string) string {
	return SiteBaseURL(domain) + "/wiki/api/v2/spaces?limit=1"
}
