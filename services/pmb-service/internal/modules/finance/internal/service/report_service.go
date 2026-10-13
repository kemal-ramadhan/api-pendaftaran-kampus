package service

import (
	"context"

	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/repository"
)

type ReportService interface {
	GetFinancialReportSummary(ctx context.Context) (*model.FinancialReportSummaryResponse, error)
	ExportFinancialReportXLSX(ctx context.Context) ([]byte, error)
}

type reportService struct {
	reportRepo repository.ReportRepository
	exporter   exporter.ReportExporter
}

func NewReportService(reportRepo repository.ReportRepository, exp exporter.ReportExporter) ReportService {
	return &reportService{
		reportRepo: reportRepo,
		exporter:   exp,
	}
}

func (s *reportService) GetFinancialReportSummary(ctx context.Context) (*model.FinancialReportSummaryResponse, error) {
	return s.reportRepo.GetSummary(ctx)
}

func (s *reportService) ExportFinancialReportXLSX(ctx context.Context) ([]byte, error) {
	summary, err := s.reportRepo.GetSummary(ctx)
	if err != nil {
		return nil, err
	}
	invoices, err := s.reportRepo.GetAllInvoicesForExport(ctx)
	if err != nil {
		return nil, err
	}
	return s.exporter.ExportFinancialReport(summary, invoices)
}
