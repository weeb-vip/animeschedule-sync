package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

	// AnimeSchedule-first: pull its season listings (current + a few prior
	// seasons, so carryover / split-cour shows that premiered earlier are still
	// covered) and match each entry to what we already have by MAL id. Our own
	// anime_seasons classification is intentionally not consulted.
	seasons := seasonRoutes(time.Now(), seasonsLookback())
	log.Info("Fetching AnimeSchedule seasons", zap.Strings("seasons", seasons))

	var entries []animeschedule.AnimeEntry
	for _, season := range seasons {
		e, err := s.client.FetchAnimeBySeason(ctx, season)
		if err != nil {
			// Skip this season but keep going — one bad season shouldn't abort the sync.
			log.Error("Failed to fetch season, skipping", zap.String("season", season), zap.Error(err))
			continue
		}
		log.Info("Fetched season anime from AnimeSchedule", zap.String("season", season), zap.Int("count", len(e)))
		entries = append(entries, e...)
	}
	if len(entries) == 0 {
		log.Info("No season entries fetched from AnimeSchedule")
		return nil
	}

	// Extract each entry's MAL id, then batch-look-up the ones we actually have.
	type entryMatch struct {
		entry animeschedule.AnimeEntry
		malID int
	}
	var candidates []entryMatch
	malIDSet := make(map[int]struct{})
	for _, entry := range entries {
		malID, ok := parseMalID(entry.Websites.MAL)
		if !ok {
			continue
		}
		candidates = append(candidates, entryMatch{entry: entry, malID: malID})
		malIDSet[malID] = struct{}{}
	}

	malIDs := make([]int, 0, len(malIDSet))
	for id := range malIDSet {
		malIDs = append(malIDs, id)
	}

	ours, err := s.animeRepo.FindByMalIDs(malIDs)
	if err != nil {
		return fmt.Errorf("looking up anime by MAL ids: %w", err)
	}
	malIDToAnimeID := make(map[int]string, len(ours))
	for _, a := range ours {
		if a.MalID != nil {
			malIDToAnimeID[*a.MalID] = a.ID
		}
	}

	matched := 0
	for _, c := range candidates {
		animeID, ok := malIDToAnimeID[c.malID]
		if !ok {
			continue // AnimeSchedule has it, we don't — skip.
		}
		matched++

		if err := s.upsertAnimeSchedule(animeID, c.entry); err != nil {
			log.Error("Failed to upsert anime schedule",
				zap.String("animeID", animeID),
				zap.Error(err),
			)
			continue
		}

		if err := s.upsertStreamingPlatforms(animeID, c.entry.Websites.Streams); err != nil {
			log.Error("Failed to upsert streaming platforms",
				zap.String("animeID", animeID),
				zap.Error(err),
			)
		}
	}
	log.Info("Synced anime schedules",
		zap.Int("season_entries", len(entries)),
		zap.Int("with_mal_id", len(candidates)),
		zap.Int("in_our_db", len(malIDToAnimeID)),
		zap.Int("matched", matched),
	)

	return nil
}

// seasonsLookback is how many seasons BEFORE the current one to also fetch, so
// carryover / split-cour shows (which AnimeSchedule lists under their premiere
// season) still get enriched. Configurable via ANIMESCHEDULE_SEASONS_LOOKBACK
// (default 2 = current + 2 prior seasons).
func seasonsLookback() int {
	if v := os.Getenv("ANIMESCHEDULE_SEASONS_LOOKBACK"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 2
}

// seasonRoutes returns AnimeSchedule season routes for the current season plus
// `lookback` prior seasons, e.g. ["summer-2026", "spring-2026", "winter-2026"].
func seasonRoutes(t time.Time, lookback int) []string {
	routes := make([]string, 0, lookback+1)
	for i := 0; i <= lookback; i++ {
		routes = append(routes, seasonRoute(t.AddDate(0, -3*i, 0)))
	}
	return routes
}

// parseMalID extracts the MyAnimeList id from a MAL URL like
// https://myanimelist.net/anime/12345/Title.
func parseMalID(malURL string) (int, bool) {
	if malURL == "" {
		return 0, false
	}
	matches := malIDRegex.FindStringSubmatch(malURL)
	if len(matches) < 2 {
		return 0, false
	}
	id, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, false
	}
	return id, true
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
