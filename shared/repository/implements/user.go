package implements

import (
	"time"

	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user *schema.User) error {
	t := &table.UserTable{
		UID:           user.UID,
		Name:          user.Name,
		Sex:           user.Sex,
		Avatar:        user.Avatar,
		Guard:         user.Guard,
		FollowerCount: user.FollowerCount,
	}
	return r.db.Create(t).Error
}

func (r *UserRepository) CreateOrUpdateUserBatch(users []*schema.User) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, u := range users {
			var existing table.UserTable
			err := tx.Where("uid = ?", u.UID).First(&existing).Error

			if err == gorm.ErrRecordNotFound {
				t := &table.UserTable{
					UID:           u.UID,
					Name:          u.Name,
					Sex:           u.Sex,
					Avatar:        u.Avatar,
					Guard:         u.Guard,
					FollowerCount: u.FollowerCount,
				}
				if err := tx.Create(t).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				if existing.Name != u.Name {
					history := &table.UserHistoryNameTable{
						UID:  existing.UID,
						Name: existing.Name,
					}
					if err := tx.Create(history).Error; err != nil {
						return err
					}
				}

				updates := map[string]any{
					"name":           u.Name,
					"avatar":         u.Avatar,
					"guard":          u.Guard,
					"follower_count": u.FollowerCount,
				}
				if u.Sex >= 0 {
					updates["sex"] = u.Sex
				}
				if err := tx.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
			}

			if u.Medal != nil {
				medal := &table.MedalTable{
					Owner:  u.Medal.OwnerID,
					Name:   u.Medal.Name,
					Level:  u.Medal.Level,
					Target: u.Medal.TargetID,
					Color:  u.Medal.Color,
				}
				if err := tx.Where("owner = ? AND target = ?", medal.Owner, medal.Target).
					Assign(map[string]any{
						"name":  medal.Name,
						"level": medal.Level,
						"color": medal.Color,
					}).FirstOrCreate(medal).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *UserRepository) ReadUser(uid uint) (*schema.User, error) {
	var t table.UserTable
	if err := r.db.Preload("Medals").First(&t, "uid = ?", uid).Error; err != nil {
		return nil, err
	}
	return schema.ToModel(&t), nil
}

func (r *UserRepository) ReadUserBatchFresh(uids []uint, ttl time.Duration) (fresh []*schema.User, stale []uint, err error) {
	if len(uids) == 0 {
		return nil, uids, nil
	}
	var tables []table.UserTable
	if err := r.db.Preload("Medals").Where("uid IN ?", uids).Find(&tables).Error; err != nil {
		return nil, uids, err
	}
	lookup := make(map[uint]*table.UserTable, len(tables))
	for i := range tables {
		lookup[tables[i].UID] = &tables[i]
	}
	now := time.Now()
	for _, uid := range uids {
		if t, ok := lookup[uid]; ok && now.Sub(t.UpdatedAt) < ttl {
			fresh = append(fresh, schema.ToModel(t))
		} else {
			stale = append(stale, uid)
		}
	}
	return fresh, stale, nil
}

func (r *UserRepository) UpdateUser(user *schema.User) error {
	return r.db.Model(&table.UserTable{}).Where("uid = ?", user.UID).Updates(map[string]any{
		"name":           user.Name,
		"sex":            user.Sex,
		"avatar":         user.Avatar,
		"guard":          user.Guard,
		"follower_count": user.FollowerCount,
	}).Error
}

func (r *UserRepository) DeleteUser(uid uint) error {
	return r.db.Delete(&table.UserTable{}, "uid = ?", uid).Error
}
