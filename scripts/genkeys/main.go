package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outDir := "certs"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Printf("Gagal membuat direktori %s: %v\n", outDir, err)
		os.Exit(1)
	}

	privPath := filepath.Join(outDir, "jwt_private.pem")
	pubPath := filepath.Join(outDir, "jwt_public.pem")

	// Periksa apakah file sudah ada
	if _, err := os.Stat(privPath); err == nil {
		fmt.Printf("ℹ️  Kunci RSA sudah ada di %s, tidak menimpa.\n", privPath)
		return
	}

	fmt.Println("🔑 Menghasilkan pasangan kunci RSA 2048-bit untuk development...")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Gagal generate RSA key: %v\n", err)
		os.Exit(1)
	}

	// Encode private key ke PKCS#8 PEM
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		fmt.Printf("Gagal marshal private key: %v\n", err)
		os.Exit(1)
	}
	privBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}
	privFile, err := os.OpenFile(privPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Printf("Gagal membuat file private key: %v\n", err)
		os.Exit(1)
	}
	defer privFile.Close()
	if err := pem.Encode(privFile, privBlock); err != nil {
		fmt.Printf("Gagal menulis private key PEM: %v\n", err)
		os.Exit(1)
	}

	// Encode public key ke PKIX PEM
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		fmt.Printf("Gagal marshal public key: %v\n", err)
		os.Exit(1)
	}
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pubFile, err := os.OpenFile(pubPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("Gagal membuat file public key: %v\n", err)
		os.Exit(1)
	}
	defer pubFile.Close()
	if err := pem.Encode(pubFile, pubBlock); err != nil {
		fmt.Printf("Gagal menulis public key PEM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Berhasil membuat pasangan kunci RSA:\n  - Private Key: %s\n  - Public Key:  %s\n", privPath, pubPath)
}
