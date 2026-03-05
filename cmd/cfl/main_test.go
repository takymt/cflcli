package main

import "testing"

func TestNormalizeConfluenceDomain(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{
			name:  "plain domain",
			input: "example.atlassian.net",
			want:  "example.atlassian.net",
		},
		{
			name:  "https prefixed domain",
			input: "https://example.atlassian.net",
			want:  "example.atlassian.net",
		},
		{
			name:      "empty",
			input:     "",
			wantError: true,
		},
		{
			name:      "wrong host",
			input:     "example.com",
			wantError: true,
		},
		{
			name:      "contains path",
			input:     "example.atlassian.net/wiki",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeConfluenceDomain(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatal("normalizeConfluenceDomain() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeConfluenceDomain() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeConfluenceDomain() = %q, want %q", got, tt.want)
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
