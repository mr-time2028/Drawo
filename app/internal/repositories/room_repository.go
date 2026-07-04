package repositories

import (
	"drawo/internal/core/domain"
	"drawo/internal/core/ports"

	"gorm.io/gorm"
)

// RoomRepo implements ports.RoomRepository using GORM.
type RoomRepo struct {
	db *gorm.DB
}

// NewRoomRepo creates a room repository.
func NewRoomRepo(db *gorm.DB) ports.RoomRepository {
	return &RoomRepo{db: db}
}

// Insert creates a new room record.
func (r *RoomRepo) Insert(room *domain.Room) error {
	return r.db.Create(room).Error
}

// GetByID retrieves a room by ID.
func (r *RoomRepo) GetByID(id string) (*domain.Room, error) {
	var room domain.Room
	if err := r.db.First(&room, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

// Update modifies a room.
func (r *RoomRepo) Update(room *domain.Room) error {
	return r.db.Save(room).Error
}

// ListPublic returns paginated public rooms for a language.
func (r *RoomRepo) ListPublic(language string, paging domain.Paging) (*domain.PageOf[domain.Room], error) {
	var rooms []domain.Room
	var total int64

	query := r.db.Model(&domain.Room{}).Where("type = ? AND language = ?", domain.RoomTypePublic, language)
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.Limit(paging.Limit).Offset(paging.Offset).Find(&rooms).Error; err != nil {
		return nil, err
	}

	return &domain.PageOf[domain.Room]{
		Items:  rooms,
		Total:  total,
		Limit:  paging.Limit,
		Offset: paging.Offset,
	}, nil
}

// Compile-time check.
var _ ports.RoomRepository = (*RoomRepo)(nil)
