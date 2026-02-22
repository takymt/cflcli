package client

import "testing"

func TestResolveDownloadURL(t *testing.T) {
	cli := &Client{baseURL: "https://example.atlassian.net/wiki/api/v2"}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "absolute https url is preserved",
			path: "https://cdn.example.com/file.svg",
			want: "https://cdn.example.com/file.svg",
		},
		{
			name: "absolute http url is preserved",
			path: "http://cdn.example.com/file.svg",
			want: "http://cdn.example.com/file.svg",
		},
		{
			name: "wiki-prefixed path is not double-prefixed",
			path: "/wiki/download/attachments/123/logo.png",
			want: "https://example.atlassian.net/wiki/download/attachments/123/logo.png",
		},
		{
			name: "root-relative path is prefixed with wiki",
			path: "/download/attachments/123/logo.png",
			want: "https://example.atlassian.net/wiki/download/attachments/123/logo.png",
		},
		{
			name: "relative path is prefixed with wiki slash",
			path: "download/attachments/123/logo.png",
			want: "https://example.atlassian.net/wiki/download/attachments/123/logo.png",
		},
		{
			name:    "empty path returns error",
			path:    " ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cli.resolveDownloadURL(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("resolveDownloadURL() error=nil want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveDownloadURL() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveDownloadURL()=%q want %q", got, tt.want)
			}
		})
	}
}
