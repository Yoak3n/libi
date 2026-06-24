package implements

import (
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type SignedUserRepository struct {
	db *gorm.DB
}

func NewSignedUserRepository(db *gorm.DB) *SignedUserRepository {
	return &SignedUserRepository{db: db}
}

func (r *SignedUserRepository) CreateSignedUser(user *table.SignedUserTable) error {
	return r.db.Create(user).Error
}

func (r *SignedUserRepository) ReadSignedUser(uid uint) (*table.SignedUserTable, error) {
	var user table.SignedUserTable
	if err := r.db.First(&user, "uid = ?", uid).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *SignedUserRepository) UpdateSignedUser(user *table.SignedUserTable) error {
	return r.db.Save(user).Error
}

func (r *SignedUserRepository) DeleteSignedUser(uid uint) error {
	return r.db.Delete(&table.SignedUserTable{}, "uid = ?", uid).Error
}
