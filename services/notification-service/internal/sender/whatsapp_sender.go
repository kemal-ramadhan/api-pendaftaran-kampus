package sender

import (
	"context"
	"fmt"
	"log"

	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/config"
)

type WhatsAppSender interface {
	Send(ctx context.Context, phone string, message string) error
}

type waSender struct {
	cfg *config.Config
}

func NewWhatsAppSender(cfg *config.Config) WhatsAppSender {
	return &waSender{cfg: cfg}
}

func (w *waSender) Send(ctx context.Context, phone string, message string) error {
	if phone == "" {
		return fmt.Errorf("nomor telepon tujuan tidak boleh kosong")
	}

	// Sesuai PRD Bagian 4.3: Driver WA default adalah "log"
	log.Printf("[WHATSAPP-SENDER] Mengirim pesan ke %s | Pesan: %s", phone, message)
	return nil
}
