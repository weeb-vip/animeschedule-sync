package episode_air_time

import (
	"time"

	"github.com/weeb-vip/animeschedule-sync/internal/db"
	"gorm.io/gorm/clause"
)

type EpisodeAirTimeRepositoryImpl interface {
	UpsertBatch(airTimes []EpisodeAirTime) error
	FindByAnimeID(animeID string) ([]EpisodeAirTime, error)
}

type EpisodeAirTimeRepository struct {
	db *db.DB
}

func NewEpisodeAirTimeRepository(db *db.DB) EpisodeAirTimeRepositoryImpl {
	return &EpisodeAirTimeRepository{db: db}
}

func (r *EpisodeAirTimeRepository) UpsertBatch(airTimes []EpisodeAirTime) error {
	if len(airTimes) == 0 {
		return nil
	}
	for i := range airTimes {
		airTimes[i].LastSyncedAt = time.Now()
	}
	return r.db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "anime_id"}, {Name: "episode_number"}, {Name: "air_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"air_datetime", "streams_json", "last_synced_at"}),
	}).Create(&airTimes).Error
}

func (r *EpisodeAirTimeRepository) FindByAnimeID(animeID string) ([]EpisodeAirTime, error) {
	var airTimes []EpisodeAirTime
	err := r.db.DB.Where("anime_id = ?", animeID).Order("episode_number ASC, air_type ASC").Find(&airTimes).Error
	return airTimes, err
}
