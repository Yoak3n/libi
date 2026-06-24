package implements

import (
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type MedalRepository struct {
	db *gorm.DB
}

func NewMedalRepository(db *gorm.DB) *MedalRepository {
	return &MedalRepository{db: db}
}

func (r *MedalRepository) CreateMedal(medal *table.MedalTable) error {
	return r.db.Create(medal).Error
}

func (r *MedalRepository) ReadMedalsByUser(uid uint) ([]*table.MedalTable, error) {
	var medals []*table.MedalTable
	if err := r.db.Where("owner_id = ?", uid).Find(&medals).Error; err != nil {
		return nil, err
	}
	return medals, nil
}

func (r *MedalRepository) UpdateMedal(medal *table.MedalTable) error {
	return r.db.Save(medal).Error
}

func (r *MedalRepository) DeleteMedal(id uint) error {
	return r.db.Delete(&table.MedalTable{}, "id = ?", id).Error
}
