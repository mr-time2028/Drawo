package repositories

import (
	"context"

	"drawo/internal/core/domain"
	"gorm.io/gorm"
)

type ReportRepository interface {
	InsertReport(ctx context.Context, report *domain.Report) error
	GetReportByID(ctx context.Context, id string) (*domain.Report, error)
	UpdateReport(ctx context.Context, report *domain.Report) error
	ListReports(ctx context.Context, paging domain.Paging) (*domain.PageOf[domain.Report], error)
	ListReportsByStatus(ctx context.Context, status domain.ReportStatus, paging domain.Paging) (*domain.PageOf[domain.Report], error)
	CountRoundReports(ctx context.Context, roomID string, round int, reportedID string, reason domain.ReportReason) (int64, error)
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

func (r *reportRepo) GetReportByID(ctx context.Context, id string) (*domain.Report, error) {
	var report domain.Report
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *reportRepo) UpdateReport(ctx context.Context, report *domain.Report) error {
	return r.db.WithContext(ctx).Where("id = ?", report.ID).Save(report).Error
}

func (r *reportRepo) ListReports(ctx context.Context, paging domain.Paging) (*domain.PageOf[domain.Report], error) {
	return r.list(ctx, "", paging)
}

func (r *reportRepo) ListReportsByStatus(ctx context.Context, status domain.ReportStatus, paging domain.Paging) (*domain.PageOf[domain.Report], error) {
	return r.list(ctx, status, paging)
}

func (r *reportRepo) CountRoundReports(ctx context.Context, roomID string, round int, reportedID string, reason domain.ReportReason) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.Report{}).
		Where("room_id = ? AND round = ? AND reported_id = ? AND reason = ?", roomID, round, reportedID, reason).
		Count(&total).Error
	return total, err
}

func (r *reportRepo) list(ctx context.Context, status domain.ReportStatus, paging domain.Paging) (*domain.PageOf[domain.Report], error) {
	if paging.Limit <= 0 || paging.Limit > 100 {
		paging.Limit = 50
	}
	var list []domain.Report
	var total int64
	query := r.db.WithContext(ctx).Model(&domain.Report{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := query.Limit(paging.Limit).Offset(paging.Offset).Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return &domain.PageOf[domain.Report]{Items: list, Total: total, Limit: paging.Limit, Offset: paging.Offset}, nil
}
