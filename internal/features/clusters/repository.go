package clusters

import (
	"time"

	"gorm.io/gorm"
)

type Record struct {
	ID          uint       `gorm:"column:id"`
	ClusterName string     `gorm:"column:cluster_name"`
	HubName     string     `gorm:"column:hub_name"`
	Region      string     `gorm:"column:region"`
	DockNumber  string     `gorm:"column:dock_number"`
	Backlogs    int        `gorm:"column:backlogs"`
	BacklogsTS  *time.Time `gorm:"column:backlogs_ts"`
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) Available() bool {
	return r.db != nil
}

func (r Repository) List() ([]Record, error) {
	records := []Record{}
	err := r.db.Table("clusters").
		Select("id, cluster_name, COALESCE(hub_name, '') AS hub_name, region, COALESCE(dock_number, '') AS dock_number, COALESCE(backlogs, 0) AS backlogs, backlogs_ts").
		Where("COALESCE(cluster_name, '') <> ''").
		Order("cluster_name asc, hub_name asc, region asc, dock_number asc").
		Find(&records).Error
	return records, err
}
