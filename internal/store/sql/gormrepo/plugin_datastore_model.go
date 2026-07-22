package gormrepo

import "time"

type PluginDataMutationModel struct {
	PluginID       string    `gorm:"column:plugin_id;primaryKey;size:64"`
	IdempotencyKey string    `gorm:"column:idempotency_key;primaryKey;size:128"`
	RequestHash    string    `gorm:"column:request_hash;size:64;not null"`
	ResultJSON     string    `gorm:"column:result_json;type:text;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;index:idx_plugin_data_mutation_created"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (PluginDataMutationModel) TableName() string { return "sk_plugin_data_mutations" }
