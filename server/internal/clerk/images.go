package clerk

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
)

// ImageFile is a real image Clerk can accept as a profile photo or org logo.
// Emoji markers never become an ImageFile — Clerk has no emoji avatar type.
type ImageFile struct {
	Filename string
	Bytes    []byte
}

// File returns a multipart.File over the image bytes. Clerk's SDK reads
// and Closes it while building the upload.
func (f ImageFile) File() multipart.File {
	return imagePart{Reader: bytes.NewReader(f.Bytes)}
}

type imagePart struct {
	*bytes.Reader
}

func (imagePart) Close() error { return nil }

var _ multipart.File = imagePart{}
var _ io.ReadCloser = imagePart{}

// UserImages is the Clerk user profile-image plane. Tests inject a fake.
type UserImages interface {
	UpdateProfileImage(ctx context.Context, clerkUserID string, file ImageFile) error
	DeleteProfileImage(ctx context.Context, clerkUserID string) error
}
