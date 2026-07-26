package anime

import (
	"github.com/weeb-vip/animeschedule-sync/internal/db"
)

type AnimeRepositoryImpl interface {
	FindByMalIDs(malIDs []int) ([]Anime, error)
	FindByMalID(malID int) (*Anime, error)
	UpdateMalID(id string, malID int) error
}

type AnimeRepository struct {
	db *db.DB
}

func NewAnimeRepository(db *db.DB) AnimeRepositoryImpl {
	return &AnimeRepository{db: db}
}

// FindByMalIDs returns the anime we already have for the given MAL ids (batch
// lookup). This drives the AnimeSchedule-first sync: we pull a season's entries
// from AnimeSchedule and match them to what we have by MAL id, so our own
// anime_seasons classification no longer gates which anime get enriched (which
// previously dropped carryover / split-cour shows).
func (r *AnimeRepository) FindByMalIDs(malIDs []int) ([]Anime, error) {
	var animes []Anime
	if len(malIDs) == 0 {
		return animes, nil
	}
	err := r.db.DB.Where("mal_id IN ?", malIDs).Find(&animes).Error
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
