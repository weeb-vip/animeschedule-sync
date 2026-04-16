package anime_streaming

import (
	"github.com/weeb-vip/animeschedule-sync/internal/db"
	"gorm.io/gorm/clause"
)

type AnimeStreamingRepositoryImpl interface {
	UpsertBatch(platforms []AnimeStreamingPlatform) error
	FindByAnimeID(animeID string) ([]AnimeStreamingPlatform, error)
}

type AnimeStreamingRepository struct {
	db *db.DB
}

func NewAnimeStreamingRepository(db *db.DB) AnimeStreamingRepositoryImpl {
	return &AnimeStreamingRepository{db: db}
}

func (r *AnimeStreamingRepository) UpsertBatch(platforms []AnimeStreamingPlatform) error {
	if len(platforms) == 0 {
		return nil
	}
	return r.db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "anime_id"}, {Name: "platform"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "url"}),
	}).Create(&platforms).Error
}

func (r *AnimeStreamingRepository) FindByAnimeID(animeID string) ([]AnimeStreamingPlatform, error) {
	var platforms []AnimeStreamingPlatform
	err := r.db.DB.Where("anime_id = ?", animeID).Find(&platforms).Error
	return platforms, err
}
