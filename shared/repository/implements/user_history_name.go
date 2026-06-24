package implements

import (
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type UserHistoryNameRepository struct {
	db *gorm.DB
}

func NewUserHistoryNameRepository(db *gorm.DB) *UserHistoryNameRepository {
	return &UserHistoryNameRepository{db: db}
}

func (r *UserHistoryNameRepository) CreateHistoryName(name *table.UserHistoryNameTable) error {
	return r.db.Create(name).Error
}

func (r *UserHistoryNameRepository) ReadHistoryNames(uid uint) ([]*table.UserHistoryNameTable, error) {
	var names []*table.UserHistoryNameTable
	if err := r.db.Where("uid = ?", uid).Find(&names).Error; err != nil {
		return nil, err
	}
	return names, nil
}
