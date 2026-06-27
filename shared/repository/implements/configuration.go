package implements

import (
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type ConfigurationRepository struct {
	db *gorm.DB
}

func NewConfigurationRepository(db *gorm.DB) *ConfigurationRepository {
	return &ConfigurationRepository{db: db}
}

func (r *ConfigurationRepository) CreateConfiguration(conf *table.ConfigurationTable) error {
	return r.db.Create(conf).Error
}

func (r *ConfigurationRepository) ReadConfiguration(id uint) (*table.ConfigurationTable, error) {
	var conf table.ConfigurationTable
	if err := r.db.First(&conf, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &conf, nil
}

func (r *ConfigurationRepository) ReadConfigurations() ([]*table.ConfigurationTable, error) {
	var confs []*table.ConfigurationTable
	if err := r.db.Find(&confs).Error; err != nil {
		return nil, err
	}
	return confs, nil
}

func (r *ConfigurationRepository) ReadValidConfigurations() ([]*table.ConfigurationTable, error) {
	var confs []*table.ConfigurationTable
	if err := r.db.Where("invalid = ?", false).Find(&confs).Error; err != nil {
		return nil, err
	}
	return confs, nil
}

func (r *ConfigurationRepository) ReadConfigurationByType(typ string) (*table.ConfigurationTable, error) {
	var conf table.ConfigurationTable
	if err := r.db.Where("type = ?", typ).First(&conf).Error; err != nil {
		return nil, err
	}
	return &conf, nil
}

func (r *ConfigurationRepository) UpdateConfiguration(conf *table.ConfigurationTable) error {
	return r.db.Save(conf).Error
}

func (r *ConfigurationRepository) DeleteConfiguration(id uint) error {
	return r.db.Delete(&table.ConfigurationTable{}, "id = ?", id).Error
}

func (r *ConfigurationRepository) DeleteConfigurations(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Delete(&table.ConfigurationTable{}, ids).Error
}
