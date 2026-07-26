package repositories

import (
	"context"
	"drawo/internal/core/domain"
	"gorm.io/gorm"
)

type FriendRequestRepository interface {
	CreateRequest(ctx context.Context, req *domain.FriendRequest) error
	GetByID(ctx context.Context, id string) (*domain.FriendRequest, error)
	ListPending(ctx context.Context, userID string) ([]domain.FriendRequest, error)
	UpdateRequest(ctx context.Context, req *domain.FriendRequest) error
}

type friendRequestRepo struct {
	db *gorm.DB
}

func NewFriendRequestRepo(db *gorm.DB) FriendRequestRepository {
	return &friendRequestRepo{db: db}
}

func (r *friendRequestRepo) CreateRequest(ctx context.Context, req *domain.FriendRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *friendRequestRepo) GetByID(ctx context.Context, id string) (*domain.FriendRequest, error) {
	var req domain.FriendRequest
	if err := r.db.WithContext(ctx).First(&req, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *friendRequestRepo) ListPending(ctx context.Context, userID string) ([]domain.FriendRequest, error) {
	var list []domain.FriendRequest
	err := r.db.WithContext(ctx).Where("to_id = ? AND status = ?", userID, "pending").Find(&list).Error
	return list, err
}

func (r *friendRequestRepo) UpdateRequest(ctx context.Context, req *domain.FriendRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}
