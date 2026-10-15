package sync

import (
	"testing"
	"time"

	"github.com/weeb-vip/animeschedule-sync/internal/db/repositories/anime_streaming"
	"github.com/weeb-vip/animeschedule-sync/internal/services/animeschedule"
)

type fakeStreamingRepo struct {
	got []anime_streaming.AnimeStreamingPlatform
}

func (f *fakeStreamingRepo) UpsertBatch(p []anime_streaming.AnimeStreamingPlatform) error {
	f.got = p
	return nil
}

func (f *fakeStreamingRepo) FindByAnimeID(string) ([]anime_streaming.AnimeStreamingPlatform, error) {
	return nil, nil
}

func TestUpsertStreamingPlatformsKeepsFirstPerPlatform(t *testing.T) {
	repo := &fakeStreamingRepo{}
	s := &SyncService{streamingRepo: repo}

	err := s.upsertStreamingPlatforms("a1", []animeschedule.StreamEntry{
		{Platform: "crunchyroll", Name: "Crunchyroll", URL: "https://cr/us"},
		{Platform: "netflix", Name: "Netflix", URL: "https://nf"},
		{Platform: "crunchyroll", Name: "Crunchyroll", URL: "https://cr/uk"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.got) != 2 {
		t.Fatalf("want 2 platforms, got %d", len(repo.got))
	}
	if repo.got[0].URL != "https://cr/us" || repo.got[1].Platform != "netflix" {
		t.Fatalf("unexpected platforms: %+v", repo.got)
	}
}

func TestSeasonRoutesSince(t *testing.T) {
	got := seasonRoutesSince(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), 2025)
	want := []string{"fall-2026", "summer-2026", "spring-2026", "winter-2026", "fall-2025", "summer-2025", "spring-2025", "winter-2025"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v", got)
		}
	}
}
