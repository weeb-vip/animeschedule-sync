package anime_schedule

import "time"

type AnimeSchedule struct {
	ID                             string     `gorm:"column:id;primaryKey;default:UUID()" json:"id"`
	AnimeID                        string     `gorm:"column:anime_id;uniqueIndex" json:"anime_id"`
	AnimescheduleRoute             *string    `gorm:"column:animeschedule_route;null" json:"animeschedule_route"`
	JpnTime                        *time.Time `gorm:"column:jpn_time;null" json:"jpn_time"`
	SubTime                        *time.Time `gorm:"column:sub_time;null" json:"sub_time"`
	DubTime                        *time.Time `gorm:"column:dub_time;null" json:"dub_time"`
	Premier                        *time.Time `gorm:"column:premier;null" json:"premier"`
	SubPremier                     *time.Time `gorm:"column:sub_premier;null" json:"sub_premier"`
	DubPremier                     *time.Time `gorm:"column:dub_premier;null" json:"dub_premier"`
	Notes                          *string    `gorm:"column:notes;null" json:"notes"`
	DelayedTimetable               *string    `gorm:"column:delayed_timetable;null" json:"delayed_timetable"`
	SubDelayedTimetable            *string    `gorm:"column:sub_delayed_timetable;null" json:"sub_delayed_timetable"`
	DubDelayedTimetable            *string    `gorm:"column:dub_delayed_timetable;null" json:"dub_delayed_timetable"`
	EpisodeOverrideDate            *time.Time `gorm:"column:episode_override_date;null" json:"episode_override_date"`
	EpisodeOverrideEpisode         *int       `gorm:"column:episode_override_episode;null" json:"episode_override_episode"`
	EpisodeOverrideEpisodesAired   *int       `gorm:"column:episode_override_episodes_aired;null" json:"episode_override_episodes_aired"`
	SubEpisodeOverrideDate         *time.Time `gorm:"column:sub_episode_override_date;null" json:"sub_episode_override_date"`
	SubEpisodeOverrideEpisode      *int       `gorm:"column:sub_episode_override_episode;null" json:"sub_episode_override_episode"`
	SubEpisodeOverrideEpisodesAired *int      `gorm:"column:sub_episode_override_episodes_aired;null" json:"sub_episode_override_episodes_aired"`
	DubEpisodeOverrideDate         *time.Time `gorm:"column:dub_episode_override_date;null" json:"dub_episode_override_date"`
	DubEpisodeOverrideEpisode      *int       `gorm:"column:dub_episode_override_episode;null" json:"dub_episode_override_episode"`
	DubEpisodeOverrideEpisodesAired *int      `gorm:"column:dub_episode_override_episodes_aired;null" json:"dub_episode_override_episodes_aired"`
	LastSyncedAt                   time.Time  `gorm:"column:last_synced_at" json:"last_synced_at"`
	CreatedAt                      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt                      time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (AnimeSchedule) TableName() string {
	return "anime_schedule"
}
