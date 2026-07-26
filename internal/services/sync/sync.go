package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/weeb-vip/animeschedule-sync/internal/db/repositories/anime"
	"github.com/weeb-vip/animeschedule-sync/internal/db/repositories/anime_schedule"
	"github.com/weeb-vip/animeschedule-sync/internal/db/repositories/anime_streaming"
	"github.com/weeb-vip/animeschedule-sync/internal/db/repositories/episode_air_time"
	"github.com/weeb-vip/animeschedule-sync/internal/logger"
	"github.com/weeb-vip/animeschedule-sync/internal/services/animeschedule"
)

var malIDRegex = regexp.MustCompile(`/anime/(\d+)/`)

type SyncService struct {
	client             *animeschedule.Client
	animeRepo          anime.AnimeRepositoryImpl
	scheduleRepo       anime_schedule.AnimeScheduleRepositoryImpl
	streamingRepo      anime_streaming.AnimeStreamingRepositoryImpl
	episodeAirTimeRepo episode_air_time.EpisodeAirTimeRepositoryImpl
}

func NewSyncService(
	client *animeschedule.Client,
	animeRepo anime.AnimeRepositoryImpl,
	scheduleRepo anime_schedule.AnimeScheduleRepositoryImpl,
	streamingRepo anime_streaming.AnimeStreamingRepositoryImpl,
	episodeAirTimeRepo episode_air_time.EpisodeAirTimeRepositoryImpl,
) *SyncService {
	return &SyncService{
		client:             client,
		animeRepo:          animeRepo,
		scheduleRepo:       scheduleRepo,
		streamingRepo:      streamingRepo,
		episodeAirTimeRepo: episodeAirTimeRepo,
	}
}

// RunFullSync runs both the anime enrichment and timetable sync
func (s *SyncService) RunFullSync(ctx context.Context) error {
	log := logger.FromCtx(ctx)

	log.Info("Starting full sync")

	// Step 1: Enrich anime with AnimeSchedule data (discovers routes)
	if err := s.SyncAnimeDetails(ctx); err != nil {
		log.Error("Failed to sync anime details", zap.Error(err))
		// Continue with timetable sync even if this fails
	}

	// Step 2: Sync timetable for precise episode air times
	if err := s.SyncTimetable(ctx); err != nil {
		log.Error("Failed to sync timetable", zap.Error(err))
		return err
	}

	log.Info("Full sync completed")
	return nil
}

// SyncAnimeDetails fetches anime details from AnimeSchedule for airing anime with MAL IDs
func (s *SyncService) SyncAnimeDetails(ctx context.Context) error {
	log := logger.FromCtx(ctx)

	animes, err := s.animeRepo.FindCurrentlyAiringWithMalID()
	if err != nil {
		return fmt.Errorf("finding airing anime: %w", err)
	}

	log.Info("Found airing anime with MAL IDs", zap.Int("count", len(animes)))

	// Map our airing anime by MAL id so AnimeSchedule's season list can be matched
	// back to our records.
	malIDToAnimeID := make(map[int]string)
	for _, a := range animes {
		if a.MalID != nil {
			malIDToAnimeID[*a.MalID] = a.ID
		}
	}

	if len(malIDToAnimeID) == 0 {
		log.Info("No anime with MAL IDs to sync")
		return nil
	}

	// AnimeSchedule's /anime endpoint 404s on multi mal-id lookups, but serves a
	// whole season at once (each entry carries its MAL id, air times, and streams),
	// so fetch the current season and match by MAL id.
	season := seasonRoute(time.Now())
	log.Info("Fetching AnimeSchedule season", zap.String("season", season))

	entries, err := s.client.FetchAnimeBySeason(ctx, season)
	if err != nil {
		return fmt.Errorf("fetching season anime: %w", err)
	}
	log.Info("Fetched season anime from AnimeSchedule", zap.Int("count", len(entries)))

	matched := 0
	for _, entry := range entries {
		// Match by MAL URL to find our anime ID; skip AnimeSchedule anime we don't have.
		animeID := s.matchAnimeID(entry, malIDToAnimeID)
		if animeID == "" {
			continue
		}
		matched++

		if err := s.upsertAnimeSchedule(animeID, entry); err != nil {
			log.Error("Failed to upsert anime schedule",
				zap.String("animeID", animeID),
				zap.Error(err),
			)
			continue
		}

		if err := s.upsertStreamingPlatforms(animeID, entry.Websites.Streams); err != nil {
			log.Error("Failed to upsert streaming platforms",
				zap.String("animeID", animeID),
				zap.Error(err),
			)
		}
	}
	log.Info("Synced anime schedules", zap.Int("matched", matched), zap.Int("season_anime", len(entries)))

	return nil
}

// seasonRoute returns the AnimeSchedule season route for the given time, e.g.
// "summer-2026". Winter=Jan-Mar, Spring=Apr-Jun, Summer=Jul-Sep, Fall=Oct-Dec.
func seasonRoute(t time.Time) string {
	var s string
	switch t.Month() {
	case time.January, time.February, time.March:
		s = "winter"
	case time.April, time.May, time.June:
		s = "spring"
	case time.July, time.August, time.September:
		s = "summer"
	default:
		s = "fall"
	}
	return fmt.Sprintf("%s-%d", s, t.Year())
}

// SyncTimetable fetches the weekly timetable and stores precise episode air times
func (s *SyncService) SyncTimetable(ctx context.Context) error {
	log := logger.FromCtx(ctx)

	entries, err := s.client.FetchTimetable(ctx)
	if err != nil {
		return fmt.Errorf("fetching timetable: %w", err)
	}

	// Build route → animeID lookup from existing anime_schedule records
	schedules, err := s.scheduleRepo.FindAllWithRoute()
	if err != nil {
		return fmt.Errorf("finding schedules with routes: %w", err)
	}

	routeToAnimeID := make(map[string]string)
	for _, sched := range schedules {
		if sched.AnimescheduleRoute != nil {
			routeToAnimeID[*sched.AnimescheduleRoute] = sched.AnimeID
		}
	}

	log.Info("Timetable route mapping", zap.Int("knownRoutes", len(routeToAnimeID)))

	var airTimes []episode_air_time.EpisodeAirTime
	matched := 0

	for _, entry := range entries {
		animeID, ok := routeToAnimeID[entry.Route]
		if !ok {
			continue
		}
		matched++

		// Serialize streams to JSON
		var streamsJSON *string
		if len(entry.Streams) > 0 {
			data, err := json.Marshal(entry.Streams)
			if err == nil {
				s := string(data)
				streamsJSON = &s
			}
		}

		airTimes = append(airTimes, episode_air_time.EpisodeAirTime{
			AnimeID:       animeID,
			EpisodeNumber: entry.EpisodeNumber,
			AirType:       entry.AirType,
			AirDatetime:   entry.EpisodeDate,
			StreamsJSON:   streamsJSON,
		})
	}

	log.Info("Matched timetable entries",
		zap.Int("total", len(entries)),
		zap.Int("matched", matched),
		zap.Int("airTimes", len(airTimes)),
	)

	// Upsert in batches of 50
	batchSize := 50
	for i := 0; i < len(airTimes); i += batchSize {
		end := i + batchSize
		if end > len(airTimes) {
			end = len(airTimes)
		}
		if err := s.episodeAirTimeRepo.UpsertBatch(airTimes[i:end]); err != nil {
			log.Error("Failed to upsert air times batch", zap.Error(err))
		}
	}

	return nil
}

// matchAnimeID extracts the MAL ID from the AnimeSchedule entry and maps it to our anime ID
func (s *SyncService) matchAnimeID(entry animeschedule.AnimeEntry, malIDToAnimeID map[int]string) string {
	if entry.Websites.MAL != "" {
		matches := malIDRegex.FindStringSubmatch(entry.Websites.MAL)
		if len(matches) >= 2 {
			if id, err := strconv.Atoi(matches[1]); err == nil {
				if animeID, ok := malIDToAnimeID[id]; ok {
					return animeID
				}
			}
		}
	}
	return ""
}

func (s *SyncService) upsertAnimeSchedule(animeID string, entry animeschedule.AnimeEntry) error {
	schedule := &anime_schedule.AnimeSchedule{
		AnimeID:            animeID,
		AnimescheduleRoute: &entry.Route,
	}

	if !animeschedule.IsNullTime(entry.JpnTime) {
		schedule.JpnTime = &entry.JpnTime
	}
	if !animeschedule.IsNullTime(entry.SubTime) {
		schedule.SubTime = &entry.SubTime
	}
	if !animeschedule.IsNullTime(entry.DubTime) {
		schedule.DubTime = &entry.DubTime
	}
	if !animeschedule.IsNullTime(entry.Premier) {
		schedule.Premier = &entry.Premier
	}
	if !animeschedule.IsNullTime(entry.SubPremier) {
		schedule.SubPremier = &entry.SubPremier
	}
	if !animeschedule.IsNullTime(entry.DubPremier) {
		schedule.DubPremier = &entry.DubPremier
	}

	if entry.Notes != "" {
		schedule.Notes = &entry.Notes
	}
	if entry.DelayedTimetable != "" {
		schedule.DelayedTimetable = &entry.DelayedTimetable
	}
	if entry.SubDelayedTimetable != "" {
		schedule.SubDelayedTimetable = &entry.SubDelayedTimetable
	}
	if entry.DubDelayedTimetable != "" {
		schedule.DubDelayedTimetable = &entry.DubDelayedTimetable
	}

	// Episode overrides
	if !animeschedule.IsNullTime(entry.EpisodeOverride.OverrideDate) {
		schedule.EpisodeOverrideDate = &entry.EpisodeOverride.OverrideDate
		schedule.EpisodeOverrideEpisode = &entry.EpisodeOverride.OverrideEpisode
		schedule.EpisodeOverrideEpisodesAired = &entry.EpisodeOverride.EpisodesAired
	}
	if !animeschedule.IsNullTime(entry.SubEpisodeOverride.OverrideDate) {
		schedule.SubEpisodeOverrideDate = &entry.SubEpisodeOverride.OverrideDate
		schedule.SubEpisodeOverrideEpisode = &entry.SubEpisodeOverride.OverrideEpisode
		schedule.SubEpisodeOverrideEpisodesAired = &entry.SubEpisodeOverride.EpisodesAired
	}
	if !animeschedule.IsNullTime(entry.DubEpisodeOverride.OverrideDate) {
		schedule.DubEpisodeOverrideDate = &entry.DubEpisodeOverride.OverrideDate
		schedule.DubEpisodeOverrideEpisode = &entry.DubEpisodeOverride.OverrideEpisode
		schedule.DubEpisodeOverrideEpisodesAired = &entry.DubEpisodeOverride.EpisodesAired
	}

	return s.scheduleRepo.Upsert(schedule)
}

func (s *SyncService) upsertStreamingPlatforms(animeID string, streams []animeschedule.StreamEntry) error {
	if len(streams) == 0 {
		return nil
	}

	var platforms []anime_streaming.AnimeStreamingPlatform
	for _, stream := range streams {
		name := stream.Name
		platforms = append(platforms, anime_streaming.AnimeStreamingPlatform{
			AnimeID:  animeID,
			Platform: stream.Platform,
			Name:     &name,
			URL:      stream.URL,
		})
	}

	return s.streamingRepo.UpsertBatch(platforms)
}
