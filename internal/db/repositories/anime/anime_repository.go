package anime

import (
	"fmt"
	"time"

	"github.com/weeb-vip/animeschedule-sync/internal/db"
)

// currentSeason returns the season key stored in anime_seasons.season for the
// given time, e.g. "SUMMER_2026". Seasons: Winter=Jan-Mar, Spring=Apr-Jun,
// Summer=Jul-Sep, Fall=Oct-Dec.
func currentSeason(t time.Time) string {
	var s string
	switch t.Month() {
	case time.January, time.February, time.March:
		s = "WINTER"
	case time.April, time.May, time.June:
		s = "SPRING"
	case time.July, time.August, time.September:
		s = "SUMMER"
	default:
		s = "FALL"
	}
	return fmt.Sprintf("%s_%d", s, t.Year())
}

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
	season := currentSeason(time.Now())
	err := r.db.DB.
		Select("anime.*").
		Joins("JOIN anime_seasons ON anime_seasons.anime_id = anime.id").
		Where("anime.mal_id IS NOT NULL").
		Where("anime_seasons.season = ?", season).
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
