package animeschedule

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/weeb-vip/animeschedule-sync/internal/logger"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	limiter    *rate.Limiter
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		// 100 requests per minute (API allows 120, leave headroom)
		limiter: rate.NewLimiter(rate.Every(600*time.Millisecond), 1),
	}
}

func (c *Client) doRequest(ctx context.Context, req *http.Request) ([]byte, error) {
	log := logger.FromCtx(ctx)

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Warn("AnimeSchedule API error",
			zap.Int("status", resp.StatusCode),
			zap.String("url", req.URL.String()),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// FetchTimetable fetches the full weekly timetable (requires auth token)
func (c *Client) FetchTimetable(ctx context.Context) ([]TimetableEntry, error) {
	log := logger.FromCtx(ctx)

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/timetables", nil)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	body, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetching timetable: %w", err)
	}

	var entries []TimetableEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parsing timetable: %w", err)
	}

	log.Info("Fetched timetable", zap.Int("entries", len(entries)))
	return entries, nil
}

// FetchAnimeBySeason fetches all anime for a season route (e.g. "summer-2026"),
// paginating through the results. AnimeSchedule's /anime endpoint only accepts a
// single mal-id per request (multiple mal-ids 404), so we pull the whole season
// — each entry carries its MAL id and streams — and match by MAL id upstream.
func (c *Client) FetchAnimeBySeason(ctx context.Context, season string) ([]AnimeEntry, error) {
	log := logger.FromCtx(ctx)

	var allAnime []AnimeEntry
	page := 1

	for {
		u, err := url.Parse(c.baseURL + "/anime")
		if err != nil {
			return nil, err
		}

		q := u.Query()
		q.Set("seasons", season)
		q.Set("page", strconv.Itoa(page))
		u.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return nil, err
		}

		body, err := c.doRequest(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("fetching anime for season %s (page %d): %w", season, page, err)
		}

		var resp AnimeSearchResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing anime response: %w", err)
		}

		allAnime = append(allAnime, resp.Anime...)

		log.Info("Fetched season page",
			zap.String("season", season),
			zap.Int("page", page),
			zap.Int("count", len(resp.Anime)),
			zap.Int("total", resp.TotalAmount),
		)

		if len(allAnime) >= resp.TotalAmount || len(resp.Anime) == 0 {
			break
		}
		page++
	}

	return allAnime, nil
}

// FetchAnimeByRoute fetches a single anime by its AnimeSchedule route slug
func (c *Client) FetchAnimeByRoute(ctx context.Context, route string) (*AnimeEntry, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/anime/"+route, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetching anime by route %s: %w", route, err)
	}

	var entry AnimeEntry
	if err := json.Unmarshal(body, &entry); err != nil {
		return nil, fmt.Errorf("parsing anime response: %w", err)
	}

	return &entry, nil
}
