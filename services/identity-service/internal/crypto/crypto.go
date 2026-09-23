package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	ErrInvalidKey       = errors.New("kunci enkripsi harus berukuran 32 byte (AES-256)")
	ErrInvalidCiphertext = errors.New("ciphertext tidak valid atau korup")
)

// DeriveKey memastikan kunci berukuran tepat 32 bytes (AES-256).
// Menerima string hex 64-karakter atau string arbitrary yang di-hash SHA-256.
func DeriveKey(secret string) []byte {
	secret = strings.TrimSpace(secret)
	if len(secret) == 64 {
		decoded, err := hex.DecodeString(secret)
		if err == nil && len(decoded) == 32 {
			return decoded
		}
	}
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

// HashNIK menghasilkan SHA-256 hexdigest dari NIK plain untuk lookup keunikan.
func HashNIK(plainNIK string) string {
	plain := strings.TrimSpace(plainNIK)
	h := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(h[:])
}

// EncryptNIK mengenkripsi NIK plain menggunakan AES-GCM dengan nonce acak 12 byte.
// Output berupa string base64 yang memuat nonce + ciphertext + tag.
func EncryptNIK(plainNIK string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("gagal inisialisasi cipher AES: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gagal inisialisasi GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("gagal generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(strings.TrimSpace(plainNIK)), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptNIK mendekripsi ciphertext base64 kembali menjadi plain NIK.
func DecryptNIK(encryptedNIK string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKey
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encryptedNIK))
	if err != nil {
		return "", fmt.Errorf("gagal decode base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("gagal inisialisasi cipher AES: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gagal inisialisasi GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("gagal dekripsi ciphertext: %w", err)
	}

	return string(plaintext), nil
}

// MaskNIK memotong dan menyamarkan NIK untuk role selain pemilik dan admin.
// Format standar 16 digit: 4 digit awal + 8 asterisk + 4 digit akhir (mis. 3205********0001).
func MaskNIK(nik string) string {
	clean := strings.TrimSpace(nik)
	if len(clean) != 16 {
		if len(clean) <= 4 {
			return "****"
		}
		if len(clean) <= 8 {
			return clean[:2] + "****" + clean[len(clean)-2:]
		}
		prefixLen := len(clean) / 4
		suffixLen := len(clean) / 4
		maskLen := len(clean) - prefixLen - suffixLen
		return clean[:prefixLen] + strings.Repeat("*", maskLen) + clean[len(clean)-suffixLen:]
	}
	return clean[:4] + "********" + clean[12:]
}
