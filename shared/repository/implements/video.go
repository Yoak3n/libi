package implements

import (
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type VideoRepository struct {
	db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

func (r *VideoRepository) CreateVideo(video *table.VideoTable) error {
	return r.db.Create(video).Error
}

func (r *VideoRepository) ReadVideo(avid uint) (*table.VideoTable, error) {
	var video table.VideoTable
	if err := r.db.First(&video, "avid = ?", avid).Error; err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *VideoRepository) UpdateVideo(video *table.VideoTable) error {
	return r.db.Save(video).Error
}

func (r *VideoRepository) DeleteVideo(avid uint) error {
	return r.db.Delete(&table.VideoTable{}, "avid = ?", avid).Error
}
