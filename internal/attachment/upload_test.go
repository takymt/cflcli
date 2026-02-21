package attachment

import (
	"errors"
	"strings"
	"testing"
)

type fakeUpserter struct {
	calls []Asset
	errAt string
}

func (f *fakeUpserter) UpsertPageAttachment(_ string, filename, sourcePath string) error {
	if f.errAt != "" && filename == f.errAt {
		return errors.New("boom")
	}
	f.calls = append(f.calls, Asset{Filename: filename, SourcePath: sourcePath})
	return nil
}

func TestUploadPageAssets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		assets     []Asset
		errAt      string
		wantErrSub string
		wantCalls  int
	}{
		{
			name: "upload all assets",
			assets: []Asset{
				{Filename: "a.png", SourcePath: "/tmp/a.png"},
				{Filename: "b.png", SourcePath: "/tmp/b.png"},
			},
			wantCalls: 2,
		},
		{
			name: "stop on first upload error",
			assets: []Asset{
				{Filename: "a.png", SourcePath: "/tmp/a.png"},
				{Filename: "b.png", SourcePath: "/tmp/b.png"},
			},
			errAt:      "a.png",
			wantErrSub: `upload image "a.png"`,
			wantCalls:  0,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			upserter := &fakeUpserter{errAt: tc.errAt}
			err := UploadPageAssets(upserter, "123", tc.assets)
			if tc.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err != nil {
				t.Fatalf("UploadPageAssets: %v", err)
			}

			if len(upserter.calls) != tc.wantCalls {
				t.Fatalf("calls=%d want %d", len(upserter.calls), tc.wantCalls)
			}
		})
	}
}
