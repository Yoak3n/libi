package implements

import (
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type DanMuRepository struct {
	db *gorm.DB
}

func NewDanMuRepository(db *gorm.DB) *DanMuRepository {
	return &DanMuRepository{db: db}
}

func (r *DanMuRepository) CreateDanMuBatch(danmus []*table.DanMuTable) error {
	return r.db.Create(&danmus).Error
}

func (r *DanMuRepository) ReadDanMuByRoom(roomId uint, limit int) ([]*table.DanMuTable, error) {
	var danmus []*table.DanMuTable
	q := r.db.Where("room_id = ?", roomId)
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Order("created_at desc").Find(&danmus).Error; err != nil {
		return nil, err
	}
	return danmus, nil
}

func (r *DanMuRepository) ReadDanMuByUser(uid uint, limit int) ([]*table.DanMuTable, error) {
	var danmus []*table.DanMuTable
	q := r.db.Where("sender = ?", uid)
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Order("created_at desc").Find(&danmus).Error; err != nil {
		return nil, err
	}
	return danmus, nil
}
