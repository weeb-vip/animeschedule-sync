package anime

import "time"

type Anime struct {
	ID        string    `gorm:"column:id;primaryKey" json:"id"`
	MalID     *int      `gorm:"column:mal_id;null" json:"mal_id"`
	AnidbID   *string   `gorm:"column:anidbid;null" json:"anidbid"`
	Status    *string   `gorm:"column:status;null" json:"status"`
	EndDate   *string   `gorm:"column:end_date;null" json:"end_date"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Anime) TableName() string {
	return "anime"
}
