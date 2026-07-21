package streamingsite

import (
	"errors"
	"strings"
	"time"
)

const (
	MaxNameLength = 120
	MaxURLLength  = 500
)

// Language — язык/регион сайта. Используется для группировки в UI (RU / международные).
type Language string

const (
	LanguageRU    Language = "ru"
	LanguageEN    Language = "en"
	LanguageMulti Language = "multi"
)

// Ошибки домена.
var (
	ErrProfileIDRequired = errors.New("profile id is required")
	ErrNameRequired      = errors.New("name is required")
	ErrNameTooLong       = errors.New("name must be 120 characters or fewer")
	ErrURLRequired       = errors.New("url is required")
	ErrURLTooLong        = errors.New("url must be 500 characters or fewer")
	ErrInvalidLanguage   = errors.New("language must be one of: ru, en, multi")
	ErrNotFound          = errors.New("streaming site not found")
)

// StreamingSite — сайт для просмотра дорам, принадлежащий конкретному профилю.
type StreamingSite struct {
	id        int64
	profileID int64
	name      string
	url       string
	language  Language
	position  int
	createdAt time.Time
	updatedAt time.Time
}

// NewStreamingSite создаёт новый валидный StreamingSite (без сохранения в БД).
func NewStreamingSite(profileID int64, name, url string, language Language, position int) (*StreamingSite, error) {
	if profileID <= 0 {
		return nil, ErrProfileIDRequired
	}

	s := &StreamingSite{profileID: profileID, position: position}
	if err := s.SetName(name); err != nil {
		return nil, err
	}
	if err := s.SetURL(url); err != nil {
		return nil, err
	}
	if err := s.SetLanguage(language); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	s.createdAt = now
	s.updatedAt = now
	return s, nil
}

// Reconstitute восстанавливает StreamingSite из БД без валидации.
func Reconstitute(id, profileID int64, name, url string, language Language, position int, createdAt, updatedAt time.Time) *StreamingSite {
	return &StreamingSite{
		id:        id,
		profileID: profileID,
		name:      name,
		url:       url,
		language:  language,
		position:  position,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// ── Геттеры ──────────────────────────────────────────────────────────────────

func (s *StreamingSite) ID() int64            { return s.id }
func (s *StreamingSite) ProfileID() int64     { return s.profileID }
func (s *StreamingSite) Name() string         { return s.name }
func (s *StreamingSite) URL() string          { return s.url }
func (s *StreamingSite) Language() Language   { return s.language }
func (s *StreamingSite) Position() int        { return s.position }
func (s *StreamingSite) CreatedAt() time.Time { return s.createdAt }
func (s *StreamingSite) UpdatedAt() time.Time { return s.updatedAt }

// ── Сеттеры ──────────────────────────────────────────────────────────────────

func (s *StreamingSite) SetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	if len([]rune(name)) > MaxNameLength {
		return ErrNameTooLong
	}
	s.name = name
	s.touch()
	return nil
}

func (s *StreamingSite) SetURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return ErrURLRequired
	}
	if len(url) > MaxURLLength {
		return ErrURLTooLong
	}
	s.url = url
	s.touch()
	return nil
}

func (s *StreamingSite) SetLanguage(language Language) error {
	switch language {
	case LanguageRU, LanguageEN, LanguageMulti:
		s.language = language
	default:
		return ErrInvalidLanguage
	}
	s.touch()
	return nil
}

func (s *StreamingSite) SetPosition(position int) {
	s.position = position
	s.touch()
}

func (s *StreamingSite) touch() { s.updatedAt = time.Now().UTC() }

// ParseLanguage конвертирует произвольную строку в Language, дефолт — 'ru' для пустой строки.
func ParseLanguage(raw string) (Language, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return LanguageRU, nil
	}
	switch Language(raw) {
	case LanguageRU, LanguageEN, LanguageMulti:
		return Language(raw), nil
	default:
		return "", ErrInvalidLanguage
	}
}
