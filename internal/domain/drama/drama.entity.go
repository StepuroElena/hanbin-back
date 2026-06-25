package drama

import (
	"errors"
	"strings"
	"time"
)

// ── Константы и ошибки домена ─────────────────────────────────────────────────

const (
	MaxTitleLength   = 500
	MaxGenreLength   = 100
	MaxCountryLength = 100
	MinRating        = 0.0
	MaxRating        = 10.0
	MinYear          = 1900
	MaxYear          = 2100
)

var (
	ErrTitleRequired       = errors.New("title is required")
	ErrWatchURLRequired    = errors.New("watch_url is required")
	ErrGenreRequired       = errors.New("genre is required")
	ErrCountryRequired     = errors.New("country is required")
	ErrTitleTooLong        = errors.New("title must be 500 characters or fewer")
	ErrGenreTooLong        = errors.New("genre must be 100 characters or fewer")
	ErrCountryTooLong      = errors.New("country must be 100 characters or fewer")
	ErrInvalidYear         = errors.New("release_year must be between 1900 and 2100")
	ErrInvalidRating       = errors.New("rating must be between 0 and 10")
	ErrInvalidReleaseTag   = errors.New("release_tag must be 'ongoing' or 'released'")
	ErrInvalidTranslation  = errors.New("translation_tag must be 'translated' or 'translating'")
	ErrInvalidWatchStatus  = errors.New("watch_status is invalid")
	ErrNotFound            = errors.New("drama not found")
	ErrProfileIDRequired   = errors.New("profile_id is required")
)

// ── Enum-типы ─────────────────────────────────────────────────────────────────

type ReleaseTag string

const (
	ReleaseTagOngoing  ReleaseTag = "ongoing"
	ReleaseTagReleased ReleaseTag = "released"
)

func ParseReleaseTag(s string) (ReleaseTag, error) {
	switch ReleaseTag(s) {
	case ReleaseTagOngoing, ReleaseTagReleased:
		return ReleaseTag(s), nil
	}
	return "", ErrInvalidReleaseTag
}

type TranslationTag string

const (
	TranslationTagTranslated   TranslationTag = "translated"
	TranslationTagTranslating  TranslationTag = "translating"
)

func ParseTranslationTag(s string) (TranslationTag, error) {
	switch TranslationTag(s) {
	case TranslationTagTranslated, TranslationTagTranslating:
		return TranslationTag(s), nil
	}
	return "", ErrInvalidTranslation
}

type WatchStatus string

const (
	WatchStatusPlanned   WatchStatus = "planned"
	WatchStatusWatching  WatchStatus = "watching"
	WatchStatusCompleted WatchStatus = "completed"
	WatchStatusDropped   WatchStatus = "dropped"
)

func ParseWatchStatus(s string) (WatchStatus, error) {
	switch WatchStatus(s) {
	case WatchStatusPlanned, WatchStatusWatching, WatchStatusCompleted, WatchStatusDropped:
		return WatchStatus(s), nil
	}
	return "", ErrInvalidWatchStatus
}

// ── Агрегат ───────────────────────────────────────────────────────────────────

// Drama — агрегат дорамы. Все поля приватны, доступ через конструктор и геттеры.
type Drama struct {
	id             int64
	profileID      int64
	title          string
	watchURL       string
	releaseYear    int
	releaseTag     ReleaseTag
	translationTag TranslationTag
	genre          string
	rating         *float64 // nil = не указан
	watchStatus    WatchStatus
	country        string
	createdAt      time.Time
	updatedAt      time.Time
}

// NewDrama создаёт валидный агрегат Drama.
// watchStatus всегда принудительно устанавливается в "planned" при создании.
func NewDrama(
	profileID int64,
	title string,
	watchURL string,
	releaseYear int,
	releaseTag ReleaseTag,
	translationTag TranslationTag,
	genre string,
	rating *float64,
	country string,
) (*Drama, error) {
	if profileID <= 0 {
		return nil, ErrProfileIDRequired
	}

	d := &Drama{profileID: profileID}

	if err := d.setTitle(title); err != nil {
		return nil, err
	}
	if err := d.setWatchURL(watchURL); err != nil {
		return nil, err
	}
	if err := d.setReleaseYear(releaseYear); err != nil {
		return nil, err
	}
	d.releaseTag = releaseTag
	d.translationTag = translationTag

	if err := d.setGenre(genre); err != nil {
		return nil, err
	}
	if rating != nil {
		if err := d.setRating(*rating); err != nil {
			return nil, err
		}
	}
	if err := d.setCountry(country); err != nil {
		return nil, err
	}

	// При создании статус всегда "запланировано"
	d.watchStatus = WatchStatusPlanned

	now := time.Now().UTC()
	d.createdAt = now
	d.updatedAt = now

	return d, nil
}

// Reconstitute восстанавливает Drama из БД без валидации.
func Reconstitute(
	id, profileID int64,
	title, watchURL string,
	releaseYear int,
	releaseTag ReleaseTag,
	translationTag TranslationTag,
	genre string,
	rating *float64,
	watchStatus WatchStatus,
	country string,
	createdAt, updatedAt time.Time,
) *Drama {
	return &Drama{
		id:             id,
		profileID:      profileID,
		title:          title,
		watchURL:       watchURL,
		releaseYear:    releaseYear,
		releaseTag:     releaseTag,
		translationTag: translationTag,
		genre:          genre,
		rating:         rating,
		watchStatus:    watchStatus,
		country:        country,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

// ── Геттеры ───────────────────────────────────────────────────────────────────

func (d *Drama) ID() int64                   { return d.id }
func (d *Drama) ProfileID() int64            { return d.profileID }
func (d *Drama) Title() string               { return d.title }
func (d *Drama) WatchURL() string            { return d.watchURL }
func (d *Drama) ReleaseYear() int            { return d.releaseYear }
func (d *Drama) ReleaseTag() ReleaseTag      { return d.releaseTag }
func (d *Drama) TranslationTag() TranslationTag { return d.translationTag }
func (d *Drama) Genre() string               { return d.genre }
func (d *Drama) Rating() *float64            { return d.rating }
func (d *Drama) WatchStatus() WatchStatus    { return d.watchStatus }
func (d *Drama) Country() string             { return d.country }
func (d *Drama) CreatedAt() time.Time        { return d.createdAt }
func (d *Drama) UpdatedAt() time.Time        { return d.updatedAt }

// ── Приватные сеттеры ─────────────────────────────────────────────────────────

func (d *Drama) setTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrTitleRequired
	}
	if len([]rune(title)) > MaxTitleLength {
		return ErrTitleTooLong
	}
	d.title = title
	return nil
}

func (d *Drama) setWatchURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return ErrWatchURLRequired
	}
	d.watchURL = url
	return nil
}

func (d *Drama) setReleaseYear(year int) error {
	if year < MinYear || year > MaxYear {
		return ErrInvalidYear
	}
	d.releaseYear = year
	return nil
}

func (d *Drama) setGenre(genre string) error {
	genre = strings.TrimSpace(genre)
	if genre == "" {
		return ErrGenreRequired
	}
	if len([]rune(genre)) > MaxGenreLength {
		return ErrGenreTooLong
	}
	d.genre = genre
	return nil
}

func (d *Drama) setRating(rating float64) error {
	if rating < MinRating || rating > MaxRating {
		return ErrInvalidRating
	}
	r := rating
	d.rating = &r
	return nil
}

func (d *Drama) setCountry(country string) error {
	country = strings.TrimSpace(country)
	if country == "" {
		return ErrCountryRequired
	}
	if len([]rune(country)) > MaxCountryLength {
		return ErrCountryTooLong
	}
	d.country = country
	return nil
}
