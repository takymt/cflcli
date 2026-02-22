package client

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"reflect"
	"strings"
	"testing"
)

func TestBuildAttachmentMultipartPayload(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		file       io.Reader
		wantErrSub string
	}{
		{
			name:     "builds multipart payload",
			filename: "logo.png",
			file:     strings.NewReader("PNGDATA"),
		},
		{
			name:       "requires filename",
			filename:   " ",
			file:       strings.NewReader("PNGDATA"),
			wantErrSub: "attachment filename is required",
		},
		{
			name:       "requires reader",
			filename:   "logo.png",
			file:       nil,
			wantErrSub: "attachment file reader is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, contentType, err := buildAttachmentMultipartPayload(tt.filename, tt.file)
			if tt.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("err=%v want contains %q", err, tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildAttachmentMultipartPayload: %v", err)
			}
			if !strings.HasPrefix(contentType, "multipart/form-data;") {
				t.Fatalf("contentType=%q", contentType)
			}

			formValues, fileName, fileContent := parseMultipartPayload(t, payload, contentType)
			if fileName != "logo.png" {
				t.Fatalf("fileName=%q want %q", fileName, "logo.png")
			}
			if fileContent != "PNGDATA" {
				t.Fatalf("fileContent=%q want %q", fileContent, "PNGDATA")
			}
			if !reflect.DeepEqual(formValues, map[string]string{"minorEdit": "true"}) {
				t.Fatalf("formValues=%v want %v", formValues, map[string]string{"minorEdit": "true"})
			}
		})
	}
}

func parseMultipartPayload(t *testing.T, payload []byte, contentType string) (map[string]string, string, string) {
	t.Helper()

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("ParseMediaType: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("mediaType=%q", mediaType)
	}
	boundary := params["boundary"]
	if strings.TrimSpace(boundary) == "" {
		t.Fatalf("boundary missing in contentType=%q", contentType)
	}

	reader := multipart.NewReader(bytes.NewReader(payload), boundary)
	formValues := map[string]string{}
	var fileName string
	var fileContent string

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("NextPart: %v", err)
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("ReadAll(part): %v", err)
		}

		switch part.FormName() {
		case "file":
			fileName = part.FileName()
			fileContent = string(body)
		default:
			formValues[part.FormName()] = string(body)
		}
	}

	return formValues, fileName, fileContent
}
