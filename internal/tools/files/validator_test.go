package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePath(t *testing.T) {
	base := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid simple relative path",
			path:    "notes.txt",
			wantErr: false,
		},
		{
			name:    "valid nested relative path",
			path:    "subdir/file.txt",
			wantErr: false,
		},
		{
			name:    "reject absolute path",
			path:    "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "reject dotdot traversal",
			path:    "../escape.txt",
			wantErr: true,
		},
		{
			name:    "reject nested dotdot traversal",
			path:    "a/../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "reject dotdot in middle",
			path:    "a/../b/../../../etc/passwd",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidatePath(base, tc.path)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for path %q, got nil", tc.path)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for path %q: %v", tc.path, err)
			}
		})
	}
}

func TestValidatePath_ExistingFile(t *testing.T) {
	base := t.TempDir()

	// Create a real file.
	realFile := filepath.Join(base, "real.txt")
	if err := os.WriteFile(realFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ValidatePath(base, "real.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resolve symlinks on both sides for comparison (macOS /tmp → /private/tmp).
	wantResolved, _ := filepath.EvalSymlinks(realFile)
	if got != wantResolved {
		t.Errorf("got %q, want %q", got, wantResolved)
	}
}

func TestValidatePath_SymlinkEscape(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()

	// Create a file outside the base.
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink inside base pointing outside.
	symlinkPath := filepath.Join(base, "link.txt")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	_, err := ValidatePath(base, "link.txt")
	if err == nil {
		t.Error("expected error for symlink escaping base directory, got nil")
	}
}
