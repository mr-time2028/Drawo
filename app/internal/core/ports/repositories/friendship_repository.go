package repositories

import (
	"context"
	"drawo/internal/core/domain"
	"gorm.io/gorm"
)

type FriendshipRepository interface {
	AddFriend(ctx context.Context, friendship *domain.Friendship) error
	ListFriends(ctx context.Context, userID string) ([]domain.Friendship, error)
	RemoveFriend(ctx context.Context, userID, friendID string) error
}

type friendshipRepo struct {
	db *gorm.DB
}

func NewFriendshipRepo(db *gorm.DB) FriendshipRepository {
	return &friendshipRepo{db: db}
}

func (r *friendshipRepo) AddFriend(ctx context.Context, friendship *domain.Friendship) error {
	return r.db.WithContext(ctx).Create(friendship).Error
}

func (r *friendshipRepo) ListFriends(ctx context.Context, userID string) ([]domain.Friendship, error) {
	var list []domain.Friendship
	err := r.db.WithContext(ctx).Where("user_id = ? OR friend_id = ?", userID, userID).Find(&list).Error
	return list, err
}

func (r *friendshipRepo) RemoveFriend(ctx context.Context, userID, friendID string) error {
	return r.db.WithContext(ctx).Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", userID, friendID, friendID, userID).Delete(&domain.Friendship{}).Error
}
