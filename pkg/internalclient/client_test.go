package internalclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
)

func TestInternalClient_SuccessRequests(t *testing.T) {
	type UserDTO struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	secretToken := "secret-internal-abc"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validasi headers
		if r.Header.Get(HeaderInternalToken) != secretToken {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if r.Header.Get(HeaderRequestID) != "test-req-id" {
			http.Error(w, "Missing Request ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			if r.URL.Path == "/internal/users/123" {
				if r.URL.Query().Get("detail") == "true" {
					_ = response.WriteJSON(w, http.StatusOK, response.SuccessResponse{
						Success: true,
						Message: "Berhasil",
						Data: UserDTO{
							ID:   "123",
							Name: "Ahmad",
						},
					})
					return
				}
			}
			http.NotFound(w, r)

		case http.MethodPost:
			if r.URL.Path == "/internal/users" {
				var body UserDTO
				_ = json.NewDecoder(r.Body).Decode(&body)
				_ = response.WriteJSON(w, http.StatusCreated, response.SuccessResponse{
					Success: true,
					Message: "User dibuat",
					Data:    body,
				})
				return
			}
			http.NotFound(w, r)

		case http.MethodDelete:
			if r.URL.Path == "/internal/users/123" {
				_ = response.WriteJSON(w, http.StatusOK, response.SuccessResponse{
					Success: true,
					Message: "User dihapus",
				})
				return
			}
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		InternalToken: secretToken,
		Timeout:       2 * time.Second,
	})

	ctx := requestid.NewContext(context.Background(), "test-req-id")

	// 1. Test GET with envelope unmarshaling
	var user UserDTO
	err := client.Get(ctx, "/internal/users/123", map[string]string{"detail": "true"}, &user)
	if err != nil {
		t.Fatalf("GET gagal: %v", err)
	}
	if user.ID != "123" || user.Name != "Ahmad" {
		t.Fatalf("Data user tidak cocok: %+v", user)
	}

	// 2. Test POST
	var created UserDTO
	err = client.Post(ctx, "/internal/users", UserDTO{ID: "456", Name: "Budi"}, &created)
	if err != nil {
		t.Fatalf("POST gagal: %v", err)
	}
	if created.ID != "456" || created.Name != "Budi" {
		t.Fatalf("Created user tidak cocok: %+v", created)
	}

	// 3. Test DELETE
	err = client.Delete(ctx, "/internal/users/123", nil)
	if err != nil {
		t.Fatalf("DELETE gagal: %v", err)
	}
}

func TestInternalClient_ErrorsAndRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/retry-test" {
			attempts++
			if attempts < 2 {
				// Return 502 to trigger retry
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			_ = response.WriteJSON(w, http.StatusOK, response.SuccessResponse{
				Success: true,
				Message: "Berhasil setelah retry",
				Data:    map[string]string{"status": "ok"},
			})
			return
		}

		if r.URL.Path == "/error-404" {
			_ = response.WriteJSON(w, http.StatusNotFound, response.ErrorResponse{
				Success:   false,
				Message:   "Data tidak ditemukan",
				ErrorCode: response.ErrCodeNotFound,
			})
			return
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:    server.URL,
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
	})

	ctx := context.Background()

	// 1. Test Retry GET
	var res map[string]string
	err := client.Get(ctx, "/retry-test", nil, &res)
	if err != nil {
		t.Fatalf("GET retry harus berhasil, error: %v", err)
	}
	if res["status"] != "ok" || attempts != 2 {
		t.Fatalf("Retry tidak sesuai ekspektasi: attempts=%d, res=%v", attempts, res)
	}

	// 2. Test 404 Error Parsing
	err = client.Get(ctx, "/error-404", nil, nil)
	if err == nil {
		t.Fatal("Harus menghasilkan error")
	}
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Error harus bertipe *Error, didapat: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.ErrorCode != response.ErrCodeNotFound {
		t.Fatalf("Status atau ErrorCode tidak sesuai: %+v", apiErr)
	}
}

func TestInternalClient_PostMultipart(t *testing.T) {
	secretToken := "secret-multipart-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(HeaderInternalToken) != secretToken {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/internal/upload" {
			http.NotFound(w, r)
			return
		}

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, "Bad Form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		extra := r.FormValue("note")

		_ = response.WriteJSON(w, http.StatusOK, response.SuccessResponse{
			Success: true,
			Message: "File diunggah",
			Data: map[string]interface{}{
				"filename": header.Filename,
				"note":     extra,
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		InternalToken: secretToken,
	})

	content := "sample-image-bytes"
	var res struct {
		Filename string `json:"filename"`
		Note     string `json:"note"`
	}

	err := client.PostMultipart(
		context.Background(),
		"/internal/upload",
		"file",
		"ktp.jpg",
		strings.NewReader(content),
		map[string]string{"note": "test-note"},
		&res,
	)
	if err != nil {
		t.Fatalf("PostMultipart gagal: %v", err)
	}

	if res.Filename != "ktp.jpg" || res.Note != "test-note" {
		t.Fatalf("Hasil PostMultipart tidak cocok: %+v", res)
	}
}

