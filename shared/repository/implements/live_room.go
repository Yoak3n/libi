package implements

import (
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type LiveRoomRepository struct {
	db *gorm.DB
}

func NewLiveRoomRepository(db *gorm.DB) *LiveRoomRepository {
	return &LiveRoomRepository{db: db}
}

func (r *LiveRoomRepository) CreateLiveRoom(room *table.LiveRoomTable) error {
	return r.db.Create(room).Error
}

func (r *LiveRoomRepository) ReadLiveRoom(roomId uint) (*table.LiveRoomTable, error) {
	var room table.LiveRoomTable
	if err := r.db.First(&room, "room_id = ?", roomId).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *LiveRoomRepository) UpdateLiveRoom(room *table.LiveRoomTable) error {
	return r.db.Save(room).Error
}

func (r *LiveRoomRepository) DeleteLiveRoom(roomId uint) error {
	return r.db.Delete(&table.LiveRoomTable{}, "room_id = ?", roomId).Error
}
