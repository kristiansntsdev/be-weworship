package services

import (
	"context"
	"errors"
	"log"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/providers"
)

// ErrInvalidSongLink is returned for links that are never fetched (not http(s), odd port, credentials…).
var ErrInvalidSongLink = providers.ErrInvalidLink

// SongLinkService checks a song request's reference link: can we read a chord sheet from it?
type SongLinkService struct {
	fetcher providers.PageFetcherIface
}

func NewSongLinkService(fetcher providers.PageFetcherIface) *SongLinkService {
	return &SongLinkService{fetcher: fetcher}
}

// CheckLink fetches the page and reports whether it holds one song (with its chord sheet), a list of songs,
// or nothing usable. A page that can't be fetched is a result (status unreachable), not an error, so the
// request form can still submit the link as a plain reference.
func (s *SongLinkService) CheckLink(ctx context.Context, rawURL string) (*models.SongLinkCheck, error) {
	u, err := providers.ParseLinkURL(rawURL)
	if err != nil {
		return nil, ErrInvalidSongLink
	}

	page, err := s.fetcher.Fetch(ctx, u.String())
	if err != nil {
		res := models.SongLinkCheck{URL: u.String(), Source: sourceName(u)}
		switch {
		case errors.Is(err, providers.ErrNotHTML):
			res.Status = models.SongLinkUnsupported
			res.Reason = "This link isn't a web page."
		case errors.Is(err, providers.ErrBlockedHost), errors.Is(err, providers.ErrInvalidLink):
			res.Status = models.SongLinkUnreachable
			res.Reason = "This link can't be opened."
		default:
			log.Printf("[song-link] fetch %s failed: %v", res.Source, err)
			res.Status = models.SongLinkUnreachable
			res.Reason = "We couldn't open this page. Check the link and try again."
		}
		return &res, nil
	}

	res := ParseSongPage(page.URL, page.HTML)
	return &res, nil
}
