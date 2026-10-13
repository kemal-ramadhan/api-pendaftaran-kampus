package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

// GenerateSecureOTP menghasilkan kode OTP 6 digit acak menggunakan crypto/rand
func GenerateSecureOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	code := int(n.Int64()) + 100000
	return fmt.Sprintf("%06d", code), nil
}

// HashOTP menghasilkan SHA-256 hash dari kode OTP dan secret key
func HashOTP(code, secret string) string {
	hasher := sha256.New()
	hasher.Write([]byte(code + ":" + secret))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GenerateResetToken menghasilkan token 32-byte acak base64url menggunakan crypto/rand
func GenerateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken menghasilkan SHA-256 hash dari token reset password
func HashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}

// NormalizePhone menormalkan format nomor telepon Indonesia menjadi +62...
func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}

	// Hapus karakter non-digit kecuali + di awal
	var clean strings.Builder
	for i, r := range phone {
		if i == 0 && r == '+' {
			clean.WriteRune(r)
		} else if r >= '0' && r <= '9' {
			clean.WriteRune(r)
		}
	}
	res := clean.String()

	if strings.HasPrefix(res, "08") {
		res = "+628" + res[2:]
	} else if strings.HasPrefix(res, "628") {
		res = "+" + res
	}

	return res
}
