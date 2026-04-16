package anime

import (
	"github.com/weeb-vip/animeschedule-sync/internal/db"
)

type AnimeRepositoryImpl interface {
	FindCurrentlyAiringWithMalID() ([]Anime, error)
	FindByMalID(malID int) (*Anime, error)
	UpdateMalID(id string, malID int) error
}

type AnimeRepository struct {
	db *db.DB
}

func NewAnimeRepository(db *db.DB) AnimeRepositoryImpl {
	return &AnimeRepository{db: db}
}

func (r *AnimeRepository) FindCurrentlyAiringWithMalID() ([]Anime, error) {
	var animes []Anime
	err := r.db.DB.
		Where("mal_id IS NOT NULL").
		Where("end_date IS NULL OR end_date >= NOW()").
		Find(&animes).Error
	return animes, err
}

func (r *AnimeRepository) FindByMalID(malID int) (*Anime, error) {
	var anime Anime
	err := r.db.DB.Where("mal_id = ?", malID).First(&anime).Error
	if err != nil {
		return nil, err
	}
	return &anime, nil
}

func (r *AnimeRepository) UpdateMalID(id string, malID int) error {
	return r.db.DB.Model(&Anime{}).Where("id = ?", id).Update("mal_id", malID).Error
}
