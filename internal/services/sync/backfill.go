package sync

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/weeb-vip/animeschedule-sync/internal/logger"
)

// BackfillStreaming walks every AnimeSchedule season from fromYear up to the
// current one and writes streaming platforms for each entry we have (matched
// by MAL id). The daily sync only covers recent seasons, so older shows never
// got their links; this fills them in. It deliberately writes nothing but
// streaming platforms, so schedule rows for long-finished shows stay as they
// are. Seasons are handled one at a time to keep each MAL id lookup small.
func (s *SyncService) BackfillStreaming(ctx context.Context, fromYear int) error {
	log := logger.FromCtx(ctx)

	seasons := seasonRoutesSince(time.Now(), fromYear)
	log.Info("Backfilling streaming platforms",
		zap.Int("from_year", fromYear),
		zap.Int("seasons", len(seasons)),
	)

	var fetched, matched, withStreams, failed int
	for i, season := range seasons {
		if err := ctx.Err(); err != nil {
			return err
		}

		entries, err := s.client.FetchAnimeBySeason(ctx, season)
		if err != nil {
			log.Error("Failed to fetch season, skipping", zap.String("season", season), zap.Error(err))
			continue
		}
		fetched += len(entries)

		malIDs := make([]int, 0, len(entries))
		for _, e := range entries {
			if id, ok := parseMalID(e.Websites.MAL); ok {
				malIDs = append(malIDs, id)
			}
		}
		ours, err := s.animeRepo.FindByMalIDs(malIDs)
		if err != nil {
			log.Error("Failed to look up anime by MAL ids", zap.String("season", season), zap.Error(err))
			continue
		}
		byMal := make(map[int]string, len(ours))
		for _, a := range ours {
			if a.MalID != nil {
				byMal[*a.MalID] = a.ID
			}
		}

		for _, e := range entries {
			id, ok := parseMalID(e.Websites.MAL)
			if !ok {
				continue
			}
			animeID, ok := byMal[id]
			if !ok {
				continue
			}
			matched++
			if len(e.Websites.Streams) == 0 {
				continue
			}
			if err := s.upsertStreamingPlatforms(animeID, e.Websites.Streams); err != nil {
				failed++
				log.Error("Failed to upsert streaming platforms", zap.String("animeID", animeID), zap.Error(err))
				continue
			}
			withStreams++
		}

		log.Info("Backfilled season",
			zap.String("season", season),
			zap.Int("done", i+1),
			zap.Int("of", len(seasons)),
			zap.Int("matched_so_far", matched),
			zap.Int("with_streams_so_far", withStreams),
		)
	}

	log.Info("Streaming backfill done",
		zap.Int("season_entries", fetched),
		zap.Int("matched", matched),
		zap.Int("with_streams", withStreams),
		zap.Int("failed", failed),
	)
	return nil
}

// seasonRoutesSince returns every season route from the current season back
// to winter of fromYear, newest first.
func seasonRoutesSince(now time.Time, fromYear int) []string {
	var routes []string
	// Start on the 1st so stepping back by months never rolls over (Dec 31
	// minus three months is not Sep 31).
	for t := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC); t.Year() >= fromYear; t = t.AddDate(0, -3, 0) {
		routes = append(routes, seasonRoute(t))
	}
	return routes
}
