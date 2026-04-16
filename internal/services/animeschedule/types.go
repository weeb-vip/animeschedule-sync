package animeschedule

import "time"

// AnimeSearchResponse is the paginated response from /anime?mal-ids=
type AnimeSearchResponse struct {
	Page        int          `json:"page"`
	TotalAmount int          `json:"totalAmount"`
	Anime       []AnimeEntry `json:"anime"`
}

// AnimeEntry is a single anime from the search or detail endpoint
type AnimeEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Route string `json:"route"`

	JpnTime    time.Time `json:"jpnTime"`
	SubTime    time.Time `json:"subTime"`
	DubTime    time.Time `json:"dubTime"`
	Premier    time.Time `json:"premier"`
	SubPremier time.Time `json:"subPremier"`
	DubPremier time.Time `json:"dubPremier"`

	Notes  string `json:"notes"`
	Status string `json:"status"`

	Episodes  int `json:"episodes"`
	LengthMin int `json:"lengthMin"`

	DelayedTimetable    string `json:"delayedTimetable"`
	SubDelayedTimetable string `json:"subDelayedTimetable"`
	DubDelayedTimetable string `json:"dubDelayedTimetable"`

	EpisodeOverride    EpisodeOverride `json:"episodeOverride"`
	SubEpisodeOverride EpisodeOverride `json:"subEpisodeOverride"`
	DubEpisodeOverride EpisodeOverride `json:"dubEpisodeOverride"`

	Websites Websites `json:"websites"`
	Names    Names    `json:"names"`
}

type EpisodeOverride struct {
	OverrideDate    time.Time `json:"overrideDate"`
	OverrideEpisode int       `json:"overrideEpisode"`
	EpisodesAired   int       `json:"episodesAired"`
}

type Websites struct {
	Official string          `json:"official"`
	MAL      string          `json:"mal"`
	AniList  string          `json:"aniList"`
	AniDB    string          `json:"anidb"`
	Streams  []StreamEntry   `json:"streams"`
}

type StreamEntry struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
	Name     string `json:"name"`
}

type Names struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
	Native  string `json:"native"`
}

// TimetableEntry is a single entry from the /timetables endpoint
type TimetableEntry struct {
	EpisodeDate   time.Time     `json:"episodeDate"`
	EpisodeNumber int           `json:"episodeNumber"`
	AirType       string        `json:"airType"`
	Route         string        `json:"route"`
	Title         string        `json:"title"`
	Status        string        `json:"status"`
	Streams       []StreamEntry `json:"streams"`
	LengthMin     *int          `json:"lengthMin,omitempty"`
	Episodes      *int          `json:"episodes,omitempty"`
}

// NullTime is the sentinel zero time from the API meaning "not set"
var NullTime = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)

// IsNullTime checks if a time is the API's null sentinel
func IsNullTime(t time.Time) bool {
	return t.Equal(NullTime)
}
