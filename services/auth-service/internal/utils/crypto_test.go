package utils

import (
	"strconv"
	"strings"
	"testing"
)

func TestGenerateSecureOTP(t *testing.T) {
	// Memastikan OTP yang dihasilkan selalu 6 digit angka
	for i := 0; i < 50; i++ {
		otp, err := GenerateSecureOTP()
		if err != nil {
			t.Fatalf("GenerateSecureOTP error: %v", err)
		}

		if len(otp) != 6 {
			t.Errorf("Panjang OTP harus 6 digit, didapat: %d (otp: %s)", len(otp), otp)
		}

		num, err := strconv.Atoi(otp)
		if err != nil {
			t.Errorf("OTP harus berupa numerik murni, error: %v", err)
		}

		if num < 100000 || num > 999999 {
			t.Errorf("OTP harus berada pada rentang 100000-999999, didapat: %d", num)
		}
	}
}

func TestHashOTP(t *testing.T) {
	secret := "test-secret-key-12345"
	otp := "123456"

	hash1 := HashOTP(otp, secret)
	hash2 := HashOTP(otp, secret)

	if hash1 == "" {
		t.Fatal("Hash OTP tidak boleh kosong")
	}

	// Deterministic
	if hash1 != hash2 {
		t.Errorf("Hash OTP harus deterministik untuk input dan secret yang sama")
	}

	// Celah No. 3: Plaintext OTP tidak boleh muncul dalam hash
	if strings.Contains(hash1, otp) {
		t.Errorf("Hash OTP tidak boleh memuat plaintext OTP")
	}

	// Secret berbeda menghasilkan hash berbeda
	hashDiffSecret := HashOTP(otp, "different-secret")
	if hash1 == hashDiffSecret {
		t.Errorf("Secret berbeda harus menghasilkan hash berbeda")
	}
}

func TestGenerateResetTokenAndHash(t *testing.T) {
	token1, err := GenerateResetToken()
	if err != nil {
		t.Fatalf("GenerateResetToken error: %v", err)
	}

	token2, err := GenerateResetToken()
	if err != nil {
		t.Fatalf("GenerateResetToken error: %v", err)
	}

	if token1 == token2 {
		t.Errorf("Dua reset token berturutan tidak boleh identik")
	}

	if len(token1) < 32 {
		t.Errorf("Panjang reset token minimal 32 karakter, didapat %d", len(token1))
	}

	hash1 := HashToken(token1)
	hash2 := HashToken(token1)

	if hash1 != hash2 {
		t.Errorf("HashToken harus deterministik")
	}

	// Celah No. 3: Plaintext token tidak boleh muncul dalam hash
	if strings.Contains(hash1, token1) {
		t.Errorf("HashToken tidak boleh memuat plaintext token")
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"081234567890", "+6281234567890"},
		{"6281234567890", "+6281234567890"},
		{"+6281234567890", "+6281234567890"},
		{"0812-3456-7890", "+6281234567890"},
		{"  0812 3456 7890  ", "+6281234567890"},
		{"", ""},
	}

	for _, tt := range tests {
		got := NormalizePhone(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizePhone(%q) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}
