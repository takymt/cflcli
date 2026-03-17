package main

import "testing"

func TestNewHTTPClientBuildsBaseURLFromRawDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{
			name:   "plain host",
			domain: "example.atlassian.net",
			want:   "https://example.atlassian.net",
		},
		{
			name:   "url with wiki path",
			domain: "https://example.atlassian.net/wiki",
			want:   "https://example.atlassian.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := newHTTPClient(clientConfig{
				domain: tt.domain,
				email:  "user@example.com",
				token:  "secret",
			}, nil)
			if err != nil {
				t.Fatalf("newHTTPClient() error = %v", err)
			}

			httpClient, ok := client.(*httpClient)
			if !ok {
				t.Fatalf("newHTTPClient() type = %T, want *httpClient", client)
			}
			if httpClient.siteBaseURL != tt.want {
				t.Fatalf("siteBaseURL = %q, want %q", httpClient.siteBaseURL, tt.want)
			}
		})
	}
}

func TestToPageURL(t *testing.T) {
	t.Parallel()

	client := &httpClient{siteBaseURL: "https://example.atlassian.net"}

	t.Run("uses webui path under wiki", func(t *testing.T) {
		t.Parallel()

		api := &apiPage{ID: "123"}
		api.Links.WebUI = "/spaces/TEST/pages/123"
		got := client.toPage(api)
		want := "https://example.atlassian.net/wiki/spaces/TEST/pages/123"
		if got.URL != want {
			t.Fatalf("toPage().URL = %q, want %q", got.URL, want)
		}
	})

	t.Run("uses absolute webui url", func(t *testing.T) {
		t.Parallel()

		api := &apiPage{ID: "123"}
		api.Links.WebUI = "https://example.atlassian.net/wiki/spaces/TEST/pages/123"
		got := client.toPage(api)
		if got.URL != api.Links.WebUI {
			t.Fatalf("toPage().URL = %q, want %q", got.URL, api.Links.WebUI)
		}
	})

	t.Run("falls back to legacy viewpage url", func(t *testing.T) {
		t.Parallel()

		api := &apiPage{ID: "123"}
		got := client.toPage(api)
		want := "https://example.atlassian.net/wiki/pages/viewpage.action?pageId=123"
		if got.URL != want {
			t.Fatalf("toPage().URL = %q, want %q", got.URL, want)
		}
	})
}
