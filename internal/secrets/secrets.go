package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sammcj/skint/internal/config"
	"github.com/zalando/go-keyring"
)

// ServiceName is the keyring service name
const ServiceName = "skint"

// Storage type constants for API key references
const (
	StorageTypeKeyring = "keyring"
	StorageTypeFile    = "file"
)

// Manager handles secure storage of API keys
type Manager struct {
	useKeyring bool
	dataDir    string
	fileStore  *FileStore
}

// NewManager creates a new secrets manager
func NewManager() (*Manager, error) {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	// Create data directory if needed
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Test if keyring is available
	useKeyring := testKeyring()

	m := &Manager{
		useKeyring: useKeyring,
		dataDir:    dataDir,
	}

	if !useKeyring {
		// Initialize file-based store
		fileStore, err := NewFileStore(dataDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create file store: %w", err)
		}
		m.fileStore = fileStore
	}

	return m, nil
}

// testKeyring tests if the OS keyring is available by probing for a
// non-existent key. ErrNotFound means the keyring works; any other error
// means it's unavailable.
func testKeyring() bool {
	_, err := keyring.Get(ServiceName, "skint_probe_nonexistent")
	return err == keyring.ErrNotFound
}

// IsKeyringAvailable returns true if the OS keyring is being used
func (m *Manager) IsKeyringAvailable() bool {
	return m.useKeyring
}

// Store saves an API key securely
func (m *Manager) Store(providerName, apiKey string) error {
	if m.useKeyring {
		return keyring.Set(ServiceName, providerName, apiKey)
	}
	return m.fileStore.Store(providerName, apiKey)
}

// Retrieve retrieves an API key
func (m *Manager) Retrieve(providerName string) (string, error) {
	if m.useKeyring {
		return keyring.Get(ServiceName, providerName)
	}
	return m.fileStore.Retrieve(providerName)
}

// Delete removes an API key
func (m *Manager) Delete(providerName string) error {
	if m.useKeyring {
		return keyring.Delete(ServiceName, providerName)
	}
	return m.fileStore.Delete(providerName)
}

// StoreWithReference stores a key and returns the reference string
func (m *Manager) StoreWithReference(providerName, apiKey string) (string, error) {
	if err := m.Store(providerName, apiKey); err != nil {
		return "", err
	}

	if m.useKeyring {
		return fmt.Sprintf("%s:%s", StorageTypeKeyring, providerName), nil
	}
	return fmt.Sprintf("%s:%s", StorageTypeFile, providerName), nil
}

// RetrieveByReference retrieves a key using a reference string
func (m *Manager) RetrieveByReference(ref string) (string, error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid reference format: %s", ref)
	}

	refType := parts[0]
	providerName := parts[1]

	switch refType {
	case StorageTypeKeyring:
		// Always try keyring first for keyring references
		return keyring.Get(ServiceName, providerName)
	case StorageTypeFile:
		// Use file store
		if m.fileStore == nil {
			return "", fmt.Errorf("file store not initialized")
		}
		return m.fileStore.Retrieve(providerName)
	default:
		return "", fmt.Errorf("unknown reference type: %s", refType)
	}
}

// MigrateFromOld migrates API keys from the old secrets.env format
func (m *Manager) MigrateFromOld() (map[string]string, error) {
	migration, err := config.NewMigration()
	if err != nil {
		return nil, err
	}
	if !migration.HasOldInstallation() {
		return nil, nil // No old installation
	}

	_, keys, err := migration.Import()
	if err != nil {
		return nil, fmt.Errorf("failed to import old secrets: %w", err)
	}

	// Store all keys
	stored := make(map[string]string)
	for providerName, apiKey := range keys {
		if err := m.Store(providerName, apiKey); err != nil {
			return stored, fmt.Errorf("failed to store key for %s: %w", providerName, err)
		}
		stored[providerName] = apiKey
	}

	return stored, nil
}

// CleanupOld removes old installation files
func (m *Manager) CleanupOld() error {
	migration, err := config.NewMigration()
	if err != nil {
		return err
	}
	return migration.Cleanup()
}

// FileStore is a file-based encrypted store for API keys
type FileStore struct {
	dataDir string
	cipher  *Cipher
}

// NewFileStore creates a new file-based store
func NewFileStore(dataDir string) (*FileStore, error) {
	cipher, err := NewCipher(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return &FileStore{
		dataDir: dataDir,
		cipher:  cipher,
	}, nil
}

// Store saves an API key to the encrypted file
func (fs *FileStore) Store(providerName, apiKey string) error {
	// Load existing secrets
	secrets, err := fs.loadAll()
	if err != nil {
		return err
	}

	// Update
	secrets[providerName] = apiKey

	// Save
	return fs.saveAll(secrets)
}

// Retrieve retrieves an API key from the encrypted file
func (fs *FileStore) Retrieve(providerName string) (string, error) {
	secrets, err := fs.loadAll()
	if err != nil {
		return "", err
	}

	key, ok := secrets[providerName]
	if !ok {
		return "", fmt.Errorf("no API key found for %s", providerName)
	}

	return key, nil
}

// Delete removes an API key from the encrypted file
func (fs *FileStore) Delete(providerName string) error {
	secrets, err := fs.loadAll()
	if err != nil {
		return err
	}

	delete(secrets, providerName)
	return fs.saveAll(secrets)
}

// secretsFile returns the path to the encrypted secrets file
func (fs *FileStore) secretsFile() string {
	return filepath.Join(fs.dataDir, "secrets.enc")
}

// loadAll loads all secrets from the encrypted file
func (fs *FileStore) loadAll() (map[string]string, error) {
	file := fs.secretsFile()

	// Check if file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	// Check for symlink
	info, err := os.Lstat(file)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("secrets file is a symlink - refusing for security")
	}

	// Read encrypted data
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets file: %w", err)
	}

	// Decrypt
	decrypted, err := fs.cipher.Decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secrets: %w", err)
	}

	// Parse (simple format: key=value per line, with basic escaping)
	secrets := make(map[string]string)
	lines := strings.Split(string(decrypted), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Unescape \n and \=
			value = strings.ReplaceAll(value, "\\=", "=")
			value = strings.ReplaceAll(value, "\\n", "\n")
			secrets[key] = value
		}
	}

	return secrets, nil
}

// saveAll saves all secrets to the encrypted file
func (fs *FileStore) saveAll(secrets map[string]string) error {
	// Sort keys for deterministic output
	names := make([]string, 0, len(secrets))
	for name := range secrets {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build content
	var lines []string
	for _, name := range names {
		key := secrets[name]
		// Basic escaping for = and newlines in values
		safeKey := strings.ReplaceAll(key, "\n", "\\n")
		safeKey = strings.ReplaceAll(safeKey, "=", "\\=")
		lines = append(lines, fmt.Sprintf("%s=%s", name, safeKey))
	}
	content := strings.Join(lines, "\n")

	// Encrypt
	encrypted, err := fs.cipher.Encrypt([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to encrypt secrets: %w", err)
	}

	// Write with secure permissions
	file := fs.secretsFile()
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open secrets file: %w", err)
	}

	if _, err := f.Write(encrypted); err != nil {
		f.Close()
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close secrets file: %w", err)
	}

	return nil
}
