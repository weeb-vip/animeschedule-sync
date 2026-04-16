package anime_schedule

import (
	"time"

	"github.com/weeb-vip/animeschedule-sync/internal/db"
	"gorm.io/gorm/clause"
)

type AnimeScheduleRepositoryImpl interface {
	Upsert(schedule *AnimeSchedule) error
	FindByAnimeID(animeID string) (*AnimeSchedule, error)
	FindByRoute(route string) (*AnimeSchedule, error)
	FindAllWithRoute() ([]AnimeSchedule, error)
}

type AnimeScheduleRepository struct {
	db *db.DB
}

func NewAnimeScheduleRepository(db *db.DB) AnimeScheduleRepositoryImpl {
	return &AnimeScheduleRepository{db: db}
}

func (r *AnimeScheduleRepository) Upsert(schedule *AnimeSchedule) error {
	schedule.LastSyncedAt = time.Now()
	return r.db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "anime_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"animeschedule_route", "jpn_time", "sub_time", "dub_time",
			"premier", "sub_premier", "dub_premier",
			"notes", "delayed_timetable", "sub_delayed_timetable", "dub_delayed_timetable",
			"episode_override_date", "episode_override_episode", "episode_override_episodes_aired",
			"sub_episode_override_date", "sub_episode_override_episode", "sub_episode_override_episodes_aired",
			"dub_episode_override_date", "dub_episode_override_episode", "dub_episode_override_episodes_aired",
			"last_synced_at",
		}),
	}).Create(schedule).Error
}

func (r *AnimeScheduleRepository) FindByAnimeID(animeID string) (*AnimeSchedule, error) {
	var schedule AnimeSchedule
	err := r.db.DB.Where("anime_id = ?", animeID).First(&schedule).Error
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *AnimeScheduleRepository) FindByRoute(route string) (*AnimeSchedule, error) {
	var schedule AnimeSchedule
	err := r.db.DB.Where("animeschedule_route = ?", route).First(&schedule).Error
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *AnimeScheduleRepository) FindAllWithRoute() ([]AnimeSchedule, error) {
	var schedules []AnimeSchedule
	err := r.db.DB.Where("animeschedule_route IS NOT NULL AND animeschedule_route != ''").Find(&schedules).Error
	return schedules, err
}
