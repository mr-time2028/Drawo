package repositories

import (
	"context"
	"drawo/internal/core/domain"
	"gorm.io/gorm"
)

type ReportRepository interface {
	InsertReport(ctx context.Context, report *domain.Report) error
	ListReports(ctx context.Context, paging domain.Paging) (*domain.PageOf[domain.Report], error)
}

type reportRepo struct {
	db *gorm.DB
}

func NewReportRepo(db *gorm.DB) ReportRepository {
	return &reportRepo{db: db}
}

func (r *reportRepo) InsertReport(ctx context.Context, report *domain.Report) error {
	return r.db.WithContext(ctx).Create(report).Error
}

func (r *reportRepo) ListReports(ctx context.Context, paging domain.Paging) (*domain.PageOf[domain.Report], error) {
	var list []domain.Report
	var total int64
	r.db.WithContext(ctx).Model(&domain.Report{}).Count(&total)
	r.db.WithContext(ctx).Limit(paging.Limit).Offset(paging.Offset).Order("created_at desc").Find(&list)
	return &domain.PageOf[domain.Report]{Items: list, Total: total, Limit: paging.Limit, Offset: paging.Offset}, nil
}
