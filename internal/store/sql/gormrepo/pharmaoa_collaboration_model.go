package gormrepo

import "time"

type PharmaAnnouncementModel struct {
	ID            string     `gorm:"primaryKey;size:64"`
	Kind          string     `gorm:"size:32;not null"`
	Title         string     `gorm:"size:255;not null"`
	Content       string     `gorm:"type:text;not null"`
	Status        string     `gorm:"size:32;not null;index:idx_pharma_announcements_status_published,priority:1"`
	PublishedBy   string     `gorm:"size:64;not null;default:''"`
	PublishedAt   *time.Time `gorm:"index:idx_pharma_announcements_status_published,priority:2"`
	AudienceJSON  string     `gorm:"type:text;not null"`
	DocumentsJSON string     `gorm:"type:text;not null"`
	CreatedAt     time.Time  `gorm:"not null"`
	UpdatedAt     time.Time  `gorm:"not null"`
	CreatedBy     string     `gorm:"size:64;not null;default:''"`
	UpdatedBy     string     `gorm:"size:64;not null;default:''"`
}

func (PharmaAnnouncementModel) TableName() string { return "pharma_oa_announcements" }

type PharmaColdChainRecordModel struct {
	ID                 string    `gorm:"primaryKey;size:64"`
	BalanceID          string    `gorm:"size:64;not null"`
	ProductID          string    `gorm:"size:64;not null"`
	BatchID            string    `gorm:"size:64;not null;index:idx_pharma_cold_chain_batch_time,priority:1"`
	BatchNo            string    `gorm:"size:128;not null"`
	WarehouseID        string    `gorm:"size:64;not null;index:idx_pharma_cold_chain_warehouse_time,priority:1"`
	AreaID             string    `gorm:"size:64;not null"`
	LocationID         string    `gorm:"size:64;not null"`
	TemperatureCelsius float64   `gorm:"not null"`
	HumidityPercent    float64   `gorm:"not null"`
	MinCelsius         float64   `gorm:"not null"`
	MaxCelsius         float64   `gorm:"not null"`
	RecordedAt         time.Time `gorm:"not null;index:idx_pharma_cold_chain_batch_time,priority:2;index:idx_pharma_cold_chain_warehouse_time,priority:2"`
	Source             string    `gorm:"size:64;not null"`
	PayloadJSON        string    `gorm:"type:text;not null"`
	CreatedAt          time.Time `gorm:"not null"`
	CreatedBy          string    `gorm:"size:64;not null;default:''"`
}

func (PharmaColdChainRecordModel) TableName() string { return "pharma_oa_cold_chain_records" }
