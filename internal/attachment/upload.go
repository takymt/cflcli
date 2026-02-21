package attachment

import "fmt"

// Upserter uploads or updates page attachments.
type Upserter interface {
	UpsertPageAttachment(pageID, filename, sourcePath string) error
}

// UploadPageAssets uploads assets for a page and stops at the first failure.
func UploadPageAssets(upserter Upserter, pageID string, assets []Asset) error {
	for _, asset := range assets {
		if err := upserter.UpsertPageAttachment(pageID, asset.Filename, asset.SourcePath); err != nil {
			return fmt.Errorf("upload image %q: %w", asset.Filename, err)
		}
	}
	return nil
}
