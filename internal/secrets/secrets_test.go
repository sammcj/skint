package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreStoreAndRetrieve(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	fs, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	// Store a key
	if err := fs.Store("test-provider", "sk-abc123"); err != nil {
		t.Fatalf("Store: %v", err)
	}

	// Retrieve it
	got, err := fs.Retrieve("test-provider")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if got != "sk-abc123" {
		t.Errorf("Retrieve = %q, want %q", got, "sk-abc123")
	}
}

func TestFileStoreRetrieveNonExistent(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	fs, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	_, err = fs.Retrieve("nonexistent")
	if err == nil {
		t.Error("Retrieve of nonexistent key should return error")
	}
}

func TestFileStoreOverwrite(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	fs, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	if err := fs.Store("provider", "old-key"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if err := fs.Store("provider", "new-key"); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := fs.Retrieve("provider")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if got != "new-key" {
		t.Errorf("Retrieve = %q, want %q", got, "new-key")
	}
}

func TestFileStoreDelete(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	fs, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	if err := fs.Store("provider", "some-key"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if err := fs.Delete("provider"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = fs.Retrieve("provider")
	if err == nil {
		t.Error("Retrieve after Delete should return error")
	}
}

func TestFileStoreMultipleKeys(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	fs, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	keys := map[string]string{
		"provider-a": "key-a",
		"provider-b": "key-b",
		"provider-c": "key-c",
	}

	for name, key := range keys {
		if err := fs.Store(name, key); err != nil {
			t.Fatalf("Store(%s): %v", name, err)
		}
	}

	for name, want := range keys {
		got, err := fs.Retrieve(name)
		if err != nil {
			t.Fatalf("Retrieve(%s): %v", name, err)
		}
		if got != want {
			t.Errorf("Retrieve(%s) = %q, want %q", name, got, want)
		}
	}
}

func TestFileStorePersistence(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Store with first instance
	fs1, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := fs1.Store("provider", "persistent-key"); err != nil {
		t.Fatalf("Store: %v", err)
	}

	// Retrieve with second instance
	fs2, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	got, err := fs2.Retrieve("provider")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if got != "persistent-key" {
		t.Errorf("Retrieve = %q, want %q", got, "persistent-key")
	}
}

func TestFileStoreNoLegacyKeyFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create a FileStore -- this triggers getOrCreateKey which should clean up
	// any legacy .key file
	_, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	keyFile := filepath.Join(tmpDir, ".key")
	if _, err := os.Stat(keyFile); !os.IsNotExist(err) {
		t.Error("legacy .key file should not exist after NewFileStore")
	}
}

func TestRetrieveByReferenceFormat(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		wantErr bool
	}{
		{name: "invalid format no colon", ref: "invalidref", wantErr: true},
		{name: "unknown type", ref: "unknown:provider", wantErr: true},
		{name: "empty ref", ref: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				useKeyring: false,
				dataDir:    t.TempDir(),
			}
			_, err := m.RetrieveByReference(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("RetrieveByReference(%q) error = %v, wantErr = %v", tt.ref, err, tt.wantErr)
			}
		})
	}
}
