package commands

import (
	"context"

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
	"github.com/weeb-vip/animeschedule-sync/internal/services/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Run a one-shot full sync from AnimeSchedule API",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		ctx := logger.WithCtx(context.Background(), log)

		cfg := config.LoadConfigOrPanic()
		database := db.NewDatabase(cfg.DBConfig)

		client := animeschedule.NewClient(
			cfg.AnimeScheduleConfig.BaseURL,
			cfg.AnimeScheduleConfig.Token,
		)

		syncService := sync.NewSyncService(
			client,
			animeRepo.NewAnimeRepository(database),
			scheduleRepo.NewAnimeScheduleRepository(database),
			streamingRepo.NewAnimeStreamingRepository(database),
			episodeAirTimeRepo.NewEpisodeAirTimeRepository(database),
		)

		if err := syncService.RunFullSync(ctx); err != nil {
			log.Error("Sync failed", zap.Error(err))
			return err
		}

		log.Info("Sync completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
