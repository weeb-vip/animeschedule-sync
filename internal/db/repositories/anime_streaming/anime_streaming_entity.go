package anime_streaming

import "time"

type AnimeStreamingPlatform struct {
	ID        string    `gorm:"column:id;primaryKey;default:UUID()" json:"id"`
	AnimeID   string    `gorm:"column:anime_id" json:"anime_id"`
	Platform  string    `gorm:"column:platform" json:"platform"`
	Name      *string   `gorm:"column:name;null" json:"name"`
	URL       string    `gorm:"column:url" json:"url"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (AnimeStreamingPlatform) TableName() string {
	return "anime_streaming_platform"
}
