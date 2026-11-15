package http

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"kibou/thisisyouryear/internal/domain"
)

// ClientTimezoneHeader carries the browser's IANA timezone (e.g. Europe/Kyiv)
// on every API request, so "which year is it" and the New-Year countdown follow
// each user's own midnight rather than the server's.
const ClientTimezoneHeader = "X-Client-Timezone"

// maxTimezoneNameLen bounds the header value before it reaches the tz loader.
// The longest IANA names are around 30 characters.
const maxTimezoneNameLen = 64

// SeasonConfig configures how the season (the year the app is about) is
// resolved per request. The zero value means: no preparation stage, UTC when a
// request names no valid timezone, and the real clock.
type SeasonConfig struct {
	// PreparationYear enables the preparation stage for that year (0 = off).
	PreparationYear int
	// DefaultLocation is used when the request's timezone is missing or
	// invalid. Nil means UTC.
	DefaultLocation *time.Location
	// Now is the clock; nil means time.Now.
	Now func() time.Time
}

// SeasonResolver decides the season for a request from the client's timezone
// header and the configured preparation year. Handlers that depend on "the
// current year" call Resolve once per request.
type SeasonResolver struct {
	cfg       SeasonConfig
	locations sync.Map // timezone name → *time.Location (valid names only)
}

// NewSeasonResolver builds a SeasonResolver.
func NewSeasonResolver(cfg SeasonConfig) *SeasonResolver {
	if cfg.DefaultLocation == nil {
		cfg.DefaultLocation = time.UTC
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &SeasonResolver{cfg: cfg}
}

// Resolve returns the season for the request's user. A wrong or spoofed
// timezone only shifts that user's own view by a few hours, so the header is
// trusted as-is once it parses.
func (s *SeasonResolver) Resolve(r *http.Request) domain.Season {
	return domain.SeasonAt(s.cfg.Now(), s.location(r.Header.Get(ClientTimezoneHeader)), s.cfg.PreparationYear)
}

// location loads (and caches) the named timezone, falling back to the default
// for a missing, oversized or unknown name. Only valid names are cached, so the
// cache is bounded by the size of the timezone database.
func (s *SeasonResolver) location(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxTimezoneNameLen {
		return s.cfg.DefaultLocation
	}
	if loc, ok := s.locations.Load(name); ok {
		return loc.(*time.Location)
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return s.cfg.DefaultLocation
	}
	s.locations.Store(name, loc)
	return loc
}
