package scraper

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

// HotDrama — один тайтл из блока "Горячие новинки" на главной doramatv.one.
type HotDrama struct {
	Title   string   `json:"title"`
	Link    string   `json:"link"`
	Cover   string   `json:"cover"`
	Rating  *float64 `json:"rating"`
	Genres  []string `json:"genres"`
	Ongoing bool     `json:"ongoing"`
}

const hotListSiteURL = "https://m.doramatv.one/"

// ScrapeHot возвращает список "Горячих новинок" с главной страницы doramatv.one.
// Ходит на сайт напрямую с бэкенда — не подвержено CORS-политике и лимитам
// публичных прокси (в отличие от скрейпа прямо из браузера пользователя).
func ScrapeHot(ctx context.Context, limit int) ([]HotDrama, error) {
	body, _, err := fetch(ctx, hotListSiteURL)
	if err != nil {
		return nil, err
	}

	section, ok := extractFeedSection(body, "Горячие новинки")
	if !ok {
		return nil, ErrNotFound
	}

	tiles := extractEntityCardTiles(section)
	if len(tiles) == 0 {
		return nil, ErrNotFound
	}

	if limit > 0 && len(tiles) > limit {
		tiles = tiles[:limit]
	}
	return tiles, nil
}

// extractFeedSection находит блок .feed-section, содержащий элемент с
// data-tab-text="<tabText>", и возвращает его как кусок HTML.
func extractFeedSection(body, tabText string) (string, bool) {
	marker := `data-tab-text="` + tabText + `"`
	idx := strings.Index(body, marker)
	if idx == -1 {
		return "", false
	}

	sectionStart := strings.LastIndex(body[:idx], `class="feed-section`)
	if sectionStart == -1 {
		return "", false
	}

	rest := body[sectionStart+len(`class="feed-section`):]
	nextIdx := strings.Index(rest, `class="feed-section`)
	if nextIdx == -1 {
		return body[sectionStart:], true
	}
	return body[sectionStart : sectionStart+len(`class="feed-section`)+nextIdx], true
}

var (
	reEntityTileMarker = regexp.MustCompile(`class="[^"]*entity-card-tile[^"]*"`)
	reHref             = regexp.MustCompile(`href="([^"]+)"`)
	reTileTitle        = regexp.MustCompile(`entity-card-tile__title[^>]*>([^<]+)<`)
	reImgDataOriginal  = regexp.MustCompile(`data-original="([^"]+)"`)
	reImgSrc           = regexp.MustCompile(`<img[^>]*src="([^"]+)"`)
	reCompactRate      = regexp.MustCompile(`compact-rate[^>]*title="([^"]+)"`)
	reGenre            = regexp.MustCompile(`elem_genre[^>]*>([^<]+)<`)
	rePopover          = regexp.MustCompile(`html-popover-holder[^>]*>([\s\S]{0,200}?)<`)
)

// extractEntityCardTiles разбивает секцию на карточки .entity-card-tile
// (та же стратегия "разбить по маркерам", что и в parser_doramatv.go для
// результатов поиска) и вытаскивает из каждой title/ссылку/обложку/рейтинг.
func extractEntityCardTiles(section string) []HotDrama {
	positions := reEntityTileMarker.FindAllStringIndex(section, -1)
	var out []HotDrama

	for i, pos := range positions {
		var chunk string
		if i+1 < len(positions) {
			chunk = section[pos[0]:positions[i+1][0]]
		} else {
			end := pos[0] + 4000
			if end > len(section) {
				end = len(section)
			}
			chunk = section[pos[0]:end]
		}

		hrefM := reHref.FindStringSubmatch(chunk)
		if len(hrefM) < 2 {
			continue
		}
		link := hrefM[1]
		if !strings.HasPrefix(link, "http") {
			link = "https://m.doramatv.one" + link
		}

		title := "—"
		if tm := reTileTitle.FindStringSubmatch(chunk); len(tm) >= 2 {
			title = strings.TrimSpace(tm[1])
		}

		cover := ""
		if cm := reImgDataOriginal.FindStringSubmatch(chunk); len(cm) >= 2 {
			cover = cm[1]
		} else if cm := reImgSrc.FindStringSubmatch(chunk); len(cm) >= 2 {
			cover = cm[1]
		}

		var rating *float64
		if rm := reCompactRate.FindStringSubmatch(chunk); len(rm) >= 2 {
			if v, err := strconv.ParseFloat(strings.TrimSpace(rm[1]), 64); err == nil {
				rating = &v
			}
		}

		var genres []string
		for _, gm := range reGenre.FindAllStringSubmatch(chunk, -1) {
			if len(gm) >= 2 {
				genres = append(genres, strings.TrimSpace(gm[1]))
				if len(genres) >= 2 {
					break
				}
			}
		}

		ongoing := false
		if pm := rePopover.FindStringSubmatch(chunk); len(pm) >= 2 {
			text := strings.ToLower(pm[1])
			ongoing = strings.Contains(text, "выходит") || strings.Contains(text, "аирится")
		}

		out = append(out, HotDrama{
			Title:   title,
			Link:    link,
			Cover:   cover,
			Rating:  rating,
			Genres:  genres,
			Ongoing: ongoing,
		})
	}

	return out
}
