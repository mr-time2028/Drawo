package repositories

import (
	"context"

	"gorm.io/gorm"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports"
)

// FriendRepo implements ports.FriendshipRepository and ports.FriendRequestRepository using GORM.
type FriendRepo struct {
	db *gorm.DB
}

// NewFriendRepo creates a persistent friendship repository.
func NewFriendRepo(db *gorm.DB) *FriendRepo {
	return &FriendRepo{db: db}
}

// AddFriend records an accepted friend relationship in the relational database.
func (r *FriendRepo) AddFriend(ctx context.Context, friendship *domain.Friendship) error {
	return r.db.WithContext(ctx).Create(friendship).Error
}

// ListFriends lists all accepted friendships for a user.
func (r *FriendRepo) ListFriends(ctx context.Context, userID string) ([]domain.Friendship, error) {
	var list []domain.Friendship
	if err := r.db.WithContext(ctx).Where("user_id = ? OR friend_id = ?", userID, userID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// RemoveFriend deletes a friendship record.
func (r *FriendRepo) RemoveFriend(ctx context.Context, userID, friendID string) error {
	return r.db.WithContext(ctx).Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", userID, friendID, friendID, userID).Delete(&domain.Friendship{}).Error
}

// CreateRequest creates a new pending friend request.
func (r *FriendRepo) CreateRequest(ctx context.Context, req *domain.FriendRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

// GetByID retrieves a friend request by ID.
func (r *FriendRepo) GetByID(ctx context.Context, id string) (*domain.FriendRequest, error) {
	var req domain.FriendRequest
	if err := r.db.WithContext(ctx).First(&req, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

// ListPending lists pending friend requests for a recipient.
func (r *FriendRepo) ListPending(ctx context.Context, userID string) ([]domain.FriendRequest, error) {
	var list []domain.FriendRequest
	if err := r.db.WithContext(ctx).Where("to_id = ? AND status = ?", userID, domain.FriendRequestPending).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateRequest modifies a friend request status.
func (r *FriendRepo) UpdateRequest(ctx context.Context, req *domain.FriendRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

var _ ports.FriendshipRepository = (*FriendRepo)(nil)
var _ ports.FriendRequestRepository = (*FriendRepo)(nil)
