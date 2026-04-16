package episode_air_time

import "time"

type EpisodeAirTime struct {
	ID            string    `gorm:"column:id;primaryKey;default:UUID()" json:"id"`
	AnimeID       string    `gorm:"column:anime_id" json:"anime_id"`
	EpisodeNumber int       `gorm:"column:episode_number" json:"episode_number"`
	AirType       string    `gorm:"column:air_type" json:"air_type"`
	AirDatetime   time.Time `gorm:"column:air_datetime" json:"air_datetime"`
	StreamsJSON   *string   `gorm:"column:streams_json;type:json;null" json:"streams_json"`
	LastSyncedAt  time.Time `gorm:"column:last_synced_at" json:"last_synced_at"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (EpisodeAirTime) TableName() string {
	return "episode_air_time"
}
