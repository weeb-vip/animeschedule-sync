package commands

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/weeb-vip/animeschedule-sync/config"
	"github.com/weeb-vip/animeschedule-sync/internal/db"
	animeRepo "github.com/weeb-vip/animeschedule-sync/internal/db/repositories/anime"
	scheduleRepo "github.com/weeb-vip/animeschedule-sync/internal/db/repositories/anime_schedule"
	streamingRepo "github.com/weeb-vip/animeschedule-sync/internal/db/repositories/anime_streaming"
	episodeAirTimeRepo "github.com/weeb-vip/animeschedule-sync/internal/db/repositories/episode_air_time"
	"github.com/weeb-vip/animeschedule-sync/internal/logger"
	"github.com/weeb-vip/animeschedule-sync/internal/services/animeschedule"
	syncService "github.com/weeb-vip/animeschedule-sync/internal/services/sync"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run sync on a recurring interval",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		ctx := logger.WithCtx(context.Background(), log)

		cfg := config.LoadConfigOrPanic()
		database := db.NewDatabase(cfg.DBConfig)

		interval, err := time.ParseDuration(cfg.AnimeScheduleConfig.SyncInterval)
		if err != nil {
			log.Error("Invalid sync interval", zap.String("interval", cfg.AnimeScheduleConfig.SyncInterval), zap.Error(err))
			return err
		}

		client := animeschedule.NewClient(
			cfg.AnimeScheduleConfig.BaseURL,
			cfg.AnimeScheduleConfig.Token,
		)

		svc := syncService.NewSyncService(
			client,
			animeRepo.NewAnimeRepository(database),
			scheduleRepo.NewAnimeScheduleRepository(database),
			streamingRepo.NewAnimeStreamingRepository(database),
			episodeAirTimeRepo.NewEpisodeAirTimeRepository(database),
		)

		log.Info("Starting sync service", zap.Duration("interval", interval))

		// Run immediately on start
		if err := svc.RunFullSync(ctx); err != nil {
			log.Error("Initial sync failed", zap.Error(err))
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Info("Running scheduled sync")
				if err := svc.RunFullSync(ctx); err != nil {
					log.Error("Scheduled sync failed", zap.Error(err))
				}
			case <-ctx.Done():
				log.Info("Shutting down sync service")
				return nil
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
