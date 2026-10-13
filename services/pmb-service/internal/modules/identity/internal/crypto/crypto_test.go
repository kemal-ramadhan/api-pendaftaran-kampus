package crypto

import (
	"testing"
)

func TestCrypto_EncryptDecrypt(t *testing.T) {
	key := DeriveKey("test-secret-encryption-key-for-identity-32bytes")
	if len(key) != 32 {
		t.Fatalf("Panjang key harus 32, didapat: %d", len(key))
	}

	rawNIK := "3201012345670001"
	encrypted, err := EncryptNIK(rawNIK, key)
	if err != nil {
		t.Fatalf("EncryptNIK gagal: %v", err)
	}

	if encrypted == rawNIK {
		t.Fatalf("NIK tidak boleh sama dengan ciphertext!")
	}

	decrypted, err := DecryptNIK(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptNIK gagal: %v", err)
	}

	if decrypted != rawNIK {
		t.Fatalf("Hasil dekripsi '%s' tidak cocok dengan plain '%s'", decrypted, rawNIK)
	}
}

func TestCrypto_HashNIK(t *testing.T) {
	nik := "3201012345670001"
	hash1 := HashNIK(nik)
	hash2 := HashNIK(nik)

	if hash1 != hash2 {
		t.Fatalf("HashNIK harus deterministik")
	}

	if len(hash1) != 64 {
		t.Fatalf("Panjang SHA-256 hash harus 64 hexdigest, didapat: %d", len(hash1))
	}
}

func TestCrypto_MaskNIK(t *testing.T) {
	nik16 := "3205123456780001"
	masked := MaskNIK(nik16)
	expected := "3205********0001"
	if masked != expected {
		t.Fatalf("Masking NIK 16 digit gagal: didapat '%s', diharapkan '%s'", masked, expected)
	}

	short := "12345"
	maskedShort := MaskNIK(short)
	if len(maskedShort) == 0 {
		t.Fatal("Masking short NIK tidak boleh kosong")
	}
}
