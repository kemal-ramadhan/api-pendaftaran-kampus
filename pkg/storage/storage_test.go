package storage

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func TestStorage_SniffAndValidate(t *testing.T) {
	client, err := NewClient(Config{
		Endpoint:        "127.0.0.1:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	t.Run("Valid PDF content", func(t *testing.T) {
		pdfData := []byte("%PDF-1.5\n%Header\nDummy PDF body content...")
		reader, mime, err := client.SniffAndValidate(bytes.NewReader(pdfData), []string{"application/pdf"})
		if err != nil {
			t.Fatalf("SniffAndValidate PDF failed: %v", err)
		}
		if mime != "application/pdf" {
			t.Errorf("Expected application/pdf, got %s", mime)
		}

		// Verify stream completeness
		allRead, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}
		if string(allRead) != string(pdfData) {
			t.Errorf("Reconstructed reader content mismatch")
		}
	})

	t.Run("Valid PNG image", func(t *testing.T) {
		pngHeader := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00}
		_, mime, err := client.SniffAndValidate(bytes.NewReader(pngHeader), []string{"image/png", "image/jpeg"})
		if err != nil {
			t.Fatalf("SniffAndValidate PNG failed: %v", err)
		}
		if mime != "image/png" {
			t.Errorf("Expected image/png, got %s", mime)
		}
	})

	t.Run("Disallowed file type rejected", func(t *testing.T) {
		plainText := []byte("Hello World, this is plain text.")
		_, mime, err := client.SniffAndValidate(bytes.NewReader(plainText), []string{"application/pdf", "image/png"})
		if err == nil {
			t.Fatal("Expected error for disallowed file type, got nil")
		}
		if !strings.Contains(err.Error(), "tipe file tidak diizinkan") {
			t.Errorf("Error message mismatch: %v", err)
		}
		if mime != "text/plain" {
			t.Errorf("Detected mime should be text/plain, got %s", mime)
		}
	})
}

func TestStorage_ValidationErrors(t *testing.T) {
	client, _ := NewClient(Config{
		Endpoint: "127.0.0.1:9000",
	})

	ctx := t.Context()

	if err := client.EnsureBucket(ctx, ""); err != ErrBucketEmpty {
		t.Errorf("Expected ErrBucketEmpty, got %v", err)
	}

	if err := client.Upload(ctx, "", "key", strings.NewReader(""), 0, ""); err != ErrBucketEmpty {
		t.Errorf("Expected ErrBucketEmpty, got %v", err)
	}

	if err := client.Upload(ctx, "bucket", "", strings.NewReader(""), 0, ""); err != ErrObjectKeyEmpty {
		t.Errorf("Expected ErrObjectKeyEmpty, got %v", err)
	}

	if _, err := client.Download(ctx, "", "key"); err != ErrBucketEmpty {
		t.Errorf("Expected ErrBucketEmpty, got %v", err)
	}

	if _, err := client.PresignedGetObject(ctx, "", "key", 0); err != ErrBucketEmpty {
		t.Errorf("Expected ErrBucketEmpty, got %v", err)
	}
}

func TestStorage_MinIOIntegration(t *testing.T) {
	client, err := NewClient(Config{
		Endpoint:        "127.0.0.1:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UseSSL:          false,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := t.Context()
	bucket := "pmb-admission-documents"

	// 1. Ensure bucket
	if err := client.EnsureBucket(ctx, bucket); err != nil {
		t.Skipf("Skipping integration test, MinIO not reachable: %v", err)
		return
	}

	// 2. Upload file
	testContent := "Bukti Selesai F2-M0: MinIO upload & download verified"
	objectKey := "integration-tests/test-f2-m0.txt"
	err = client.Upload(ctx, bucket, objectKey, strings.NewReader(testContent), int64(len(testContent)), "text/plain")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// 3. Download file
	reader, err := client.Download(ctx, bucket, objectKey)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer reader.Close()

	readBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(readBytes) != testContent {
		t.Fatalf("Content mismatch: expected %q, got %q", testContent, string(readBytes))
	}

	// 4. Presigned GET URL
	presignedURL, err := client.PresignedGetObject(ctx, bucket, objectKey, 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignedGetObject failed: %v", err)
	}
	if !strings.Contains(presignedURL, objectKey) {
		t.Errorf("Presigned URL does not contain object key: %s", presignedURL)
	}

	// 5. Cleanup / Delete object
	err = client.DeleteObject(ctx, bucket, objectKey)
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}
}

