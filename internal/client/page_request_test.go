package client

import (
	"reflect"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/model"
)

func TestBuildPageCreateRequest(t *testing.T) {
	tests := []struct {
		name       string
		spaceID    string
		title      string
		body       string
		parentID   string
		want       model.PageCreateRequest
		wantErrSub string
	}{
		{
			name:     "builds request and trims ids",
			spaceID:  " SPACE-1 ",
			title:    "  New Doc  ",
			body:     "<p>Hello</p>",
			parentID: " 22 ",
			want: model.PageCreateRequest{
				SpaceID:  "SPACE-1",
				Status:   "current",
				Title:    "New Doc",
				ParentID: "22",
				Body: model.PageCreateBody{
					Storage: model.BodyType{
						Representation: "storage",
						Value:          "<p>Hello</p>",
					},
				},
			},
		},
		{
			name:     "omits empty parent id",
			spaceID:  "SPACE-1",
			title:    "Doc",
			body:     "<p>Hello</p>",
			parentID: " ",
			want: model.PageCreateRequest{
				SpaceID: "SPACE-1",
				Status:  "current",
				Title:   "Doc",
				Body: model.PageCreateBody{
					Storage: model.BodyType{
						Representation: "storage",
						Value:          "<p>Hello</p>",
					},
				},
			},
		},
		{name: "missing space id", title: "Doc", wantErrSub: "space id is required"},
		{name: "missing title", spaceID: "SPACE-1", wantErrSub: "title is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildPageCreateRequest(tt.spaceID, tt.title, tt.body, tt.parentID)
			if tt.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("err=%v want contains %q", err, tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildPageCreateRequest: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("request mismatch\n got=%+v\nwant=%+v", got, tt.want)
			}
		})
	}
}

func TestBuildPageUpdateRequest(t *testing.T) {
	tests := []struct {
		name       string
		pageID     string
		title      string
		body       string
		parentID   string
		version    int
		want       model.PageUpdateRequest
		wantErrSub string
	}{
		{
			name:     "builds request and trims ids",
			pageID:   " 123 ",
			title:    " Updated Doc ",
			body:     "<p>Updated</p>",
			parentID: " 55 ",
			version:  8,
			want: model.PageUpdateRequest{
				ID:       "123",
				Status:   "current",
				Title:    "Updated Doc",
				ParentID: "55",
				Body: model.PageCreateBody{
					Storage: model.BodyType{
						Representation: "storage",
						Value:          "<p>Updated</p>",
					},
				},
				Version: model.PageUpdateRequestVersion{Number: 8},
			},
		},
		{
			name:    "omits empty parent id",
			pageID:  "123",
			title:   "Doc",
			body:    "<p>Body</p>",
			version: 1,
			want: model.PageUpdateRequest{
				ID:     "123",
				Status: "current",
				Title:  "Doc",
				Body: model.PageCreateBody{
					Storage: model.BodyType{
						Representation: "storage",
						Value:          "<p>Body</p>",
					},
				},
				Version: model.PageUpdateRequestVersion{Number: 1},
			},
		},
		{name: "missing page id", title: "Doc", version: 1, wantErrSub: "page id is required"},
		{name: "missing title", pageID: "123", version: 1, wantErrSub: "title is required"},
		{name: "invalid version", pageID: "123", title: "Doc", version: 0, wantErrSub: "version must be >= 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildPageUpdateRequest(tt.pageID, tt.title, tt.body, tt.parentID, tt.version)
			if tt.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("err=%v want contains %q", err, tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildPageUpdateRequest: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("request mismatch\n got=%+v\nwant=%+v", got, tt.want)
			}
		})
	}
}
