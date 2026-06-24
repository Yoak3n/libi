package implements

import (
	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type UserEntryRepository struct {
	db *gorm.DB
}

func NewUserEntryRepository(db *gorm.DB) *UserEntryRepository {
	return &UserEntryRepository{db: db}
}

func (r *UserEntryRepository) CreateEntry(entry *schema.UserEntry) error {
	t := &table.UserEntryTable{
		UID:       entry.UID,
		RoomId:    entry.RoomId,
		EnteredAt: entry.EnteredAt,
	}
	return r.db.Create(t).Error
}

func (r *UserEntryRepository) CreateEntryBatch(entries []*table.UserEntryTable) error {
	return r.db.Create(&entries).Error
}

func (r *UserEntryRepository) ReadEntriesByRoom(roomId uint, limit int) ([]*schema.UserEntry, error) {
	var tables []table.UserEntryTable
	if err := r.db.Where("room_id = ?", roomId).Order("entered_at DESC").Limit(limit).Find(&tables).Error; err != nil {
		return nil, err
	}
	entries := make([]*schema.UserEntry, len(tables))
	for i, t := range tables {
		entries[i] = &schema.UserEntry{
			UID:       t.UID,
			RoomId:    t.RoomId,
			EnteredAt: t.EnteredAt,
		}
	}
	return entries, nil
}

func (r *UserEntryRepository) ReadEntriesByUser(uid uint, limit int) ([]*schema.UserEntry, error) {
	var tables []table.UserEntryTable
	if err := r.db.Where("uid = ?", uid).Order("entered_at DESC").Limit(limit).Find(&tables).Error; err != nil {
		return nil, err
	}
	entries := make([]*schema.UserEntry, len(tables))
	for i, t := range tables {
		entries[i] = &schema.UserEntry{
			UID:       t.UID,
			RoomId:    t.RoomId,
			EnteredAt: t.EnteredAt,
		}
	}
	return entries, nil
}

func (r *UserEntryRepository) CountByUser(uid uint) (int64, error) {
	var count int64
	if err := r.db.Model(&table.UserEntryTable{}).Where("uid = ?", uid).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
