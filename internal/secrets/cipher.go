package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
)

// Cipher handles encryption/decryption for the file-based store
type Cipher struct {
	dataDir string
	key     []byte
}

// NewCipher creates a new cipher instance
func NewCipher(dataDir string) (*Cipher, error) {
	c := &Cipher{
		dataDir: dataDir,
	}

	// Get or create encryption key
	c.key = c.getOrCreateKey()

	return c, nil
}

// getOrCreateKey derives the encryption key from machine-specific data.
// The key is derived fresh each time (Argon2 at these params is ~50ms,
// acceptable for a CLI tool). Any legacy .key file is cleaned up.
func (c *Cipher) getOrCreateKey() []byte {
	// Clean up legacy .key file if it exists
	keyFile := filepath.Join(c.dataDir, ".key")
	_ = os.Remove(keyFile)

	return c.deriveKey()
}

// deriveKey creates a key from machine-specific data.
// The static app identifier is used as the Argon2 password and the
// machine-specific data as the salt. This is the correct orientation:
// the password is the secret component (compiled into the binary) and
// the salt provides per-machine uniqueness.
func (c *Cipher) deriveKey() []byte {
	salt := c.getMachineSalt()
	key := argon2.IDKey([]byte("skint1"), salt, 3, 64*1024, 4, 32)
	return key
}

// getMachineSalt returns machine-specific data for key derivation
func (c *Cipher) getMachineSalt() []byte {
	// Try various machine identifiers
	var components []string

	// Machine ID (Linux systemd)
	if id, err := os.ReadFile("/etc/machine-id"); err == nil {
		components = append(components, string(id))
	}

	// Hostname
	if hostname, err := os.Hostname(); err == nil {
		components = append(components, hostname)
	}

	// User home directory
	if home, err := os.UserHomeDir(); err == nil {
		components = append(components, home)
	}

	// User ID
	components = append(components, fmt.Sprintf("%d", os.Getuid()))

	// Combine all components
	combined := ""
	for _, c := range components {
		combined += c
	}

	// Hash to get consistent length
	hash := sha256.Sum256([]byte(combined))
	return hash[:]
}

// Encrypt encrypts data using AES-256-GCM
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM
func (c *Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString is a convenience method for encrypting strings
func (c *Cipher) EncryptString(plaintext string) (string, error) {
	encrypted, err := c.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString is a convenience method for decrypting strings
func (c *Cipher) DecryptString(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode: %w", err)
	}

	decrypted, err := c.Decrypt(data)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}
