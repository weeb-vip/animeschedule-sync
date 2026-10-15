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

var backfillFromYear int

var backfillStreamingCmd = &cobra.Command{
	Use:   "backfill-streaming",
	Short: "Write streaming platforms for every season back to --from-year",
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

		if err := syncService.BackfillStreaming(ctx, backfillFromYear); err != nil {
			log.Error("Streaming backfill failed", zap.Error(err))
			return err
		}
		return nil
	},
}

func init() {
	backfillStreamingCmd.Flags().IntVar(&backfillFromYear, "from-year", 1960, "oldest year to backfill")
	rootCmd.AddCommand(backfillStreamingCmd)
}
