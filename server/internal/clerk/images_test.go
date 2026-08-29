package clerk

import (
	"io"
	"testing"
)

func TestImageFileRoundTrip(t *testing.T) {
	t.Parallel()
	want := []byte{0x89, 0x50, 0x4e, 0x47}
	file := ImageFile{Filename: "avatar.png", Bytes: want}.File()
	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("bytes: got %v want %v", got, want)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
