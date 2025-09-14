package gomedia

import (
	"context"
	"io"
	"time"
)

// StorageDriver defines the basic contract for any storage backend (S3, GCS, Local, etc.).
// Implementations must handle uploading, deleting, checking existence,
// and generating URLs (public or signed).
type StorageDriver interface {
	// Delete removes a file identified by its key from storage.
	// Usage: Call when you want to permanently remove a file.
	Delete(ctx context.Context, key string) error

	// Exists checks whether a file with the given key exists in storage.
	// Returns true if found, false otherwise.
	// Usage: Useful before uploading to avoid overwriting or to verify presence.
	Exists(ctx context.Context, key string) (exists bool, err error)

	// GetSignedURL generates a temporary, time-limited URL for accessing a file.
	// Typically used for private storage where you need controlled access.
	// Usage: Call this to share a download link that expires after `expiry`.
	GetSignedURL(ctx context.Context, key string, expiry time.Duration) (url string, err error)

	// GetURL returns a direct/public URL for the file.
	// Typically used for public storage where no signing is required.
	// Usage: Call this to display or embed media that anyone can access.
	GetURL(ctx context.Context, key string) (url string, err error)

	// Put uploads a file (provided as io.Reader) to the given key in storage.
	// Returns the resulting file URL (public or internal, depending on implementation).
	// Usage: Call this to save a new file or overwrite an existing one.
	Put(ctx context.Context, file io.Reader, key string) (url string, err error)
}
