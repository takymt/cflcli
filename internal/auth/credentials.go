package auth

import (
	"errors"
	"os"
	"strings"
)

// Credentials contains the Confluence authentication values used by the CLI.
type Credentials struct {
	Domain   string `yaml:"domain"`
	Email    string `yaml:"email"`
	APIToken string `yaml:"api_token"`
}

// ResolveRuntimeCredentials loads credentials from env vars and config storage.
func ResolveRuntimeCredentials(store Store) (Credentials, error) {
	envCreds := credentialsFromEnv()
	if hasAllCredentials(envCreds) {
		return RequireCredentials(envCreds)
	}

	var configCreds Credentials
	if store != nil {
		loaded, err := store.Load()
		if err != nil {
			return Credentials{}, err
		}
		configCreds = loaded
	}

	return RequireCredentials(mergeCredentials(configCreds, envCreds))
}

// ResolveValidationCredentials merges config credentials with env overrides for validation.
func ResolveValidationCredentials(configCreds Credentials) (Credentials, error) {
	return RequireCredentials(mergeCredentials(configCreds, credentialsFromEnv()))
}

// SiteBaseURL normalizes a Confluence site base URL from a domain-like input.
func SiteBaseURL(domain string) string {
	base := strings.TrimSpace(domain)
	if base == "" {
		return ""
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "https://" + base
	}
	base = strings.TrimSuffix(base, "/")
	return strings.TrimSuffix(base, "/wiki")
}

func credentialsFromEnv() Credentials {
	return Credentials{
		Domain:   strings.TrimSpace(os.Getenv("CONFLUENCE_DOMAIN")),
		Email:    strings.TrimSpace(firstNonEmpty(os.Getenv("CONFLUENCE_EMAIL"), os.Getenv("ATLASSIAN_EMAIL"))),
		APIToken: strings.TrimSpace(firstNonEmpty(os.Getenv("CONFLUENCE_API_TOKEN"), os.Getenv("ATLASSIAN_API_TOKEN"))),
	}
}

func hasAllCredentials(creds Credentials) bool {
	return strings.TrimSpace(creds.Domain) != "" &&
		strings.TrimSpace(creds.Email) != "" &&
		strings.TrimSpace(creds.APIToken) != ""
}

func mergeCredentials(configCreds Credentials, envCreds Credentials) Credentials {
	return Credentials{
		Domain:   firstNonEmpty(envCreds.Domain, configCreds.Domain),
		Email:    firstNonEmpty(envCreds.Email, configCreds.Email),
		APIToken: firstNonEmpty(envCreds.APIToken, configCreds.APIToken),
	}
}

// RequireCredentials validates that all required credential fields are present.
func RequireCredentials(creds Credentials) (Credentials, error) {
	trimmed := Credentials{
		Domain:   strings.TrimSpace(creds.Domain),
		Email:    strings.TrimSpace(creds.Email),
		APIToken: strings.TrimSpace(creds.APIToken),
	}
	if trimmed.Domain == "" {
		return Credentials{}, errors.New("domain is required")
	}
	if trimmed.Email == "" {
		return Credentials{}, errors.New("email is required")
	}
	if trimmed.APIToken == "" {
		return Credentials{}, errors.New("api token is required")
	}
	return trimmed, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
