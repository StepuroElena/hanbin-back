package user

import (
	"errors"
	"time"
)

// Ошибки домена для дорам и бэйджей.
var (
	ErrDramaNotFound = errors.New("drama not found")
	ErrBadgeNotFound = errors.New("badge not found")
)

// WatchStatus — статус просмотра дорамы пользователем.
type WatchStatus string

const (
	WatchStatusWatching  WatchStatus = "watching"
	WatchStatusCompleted WatchStatus = "completed"
	WatchStatusPlan      WatchStatus = "plan"
	WatchStatusDropped   WatchStatus = "dropped"
)

// Drama — агрегат дорамы в списке пользователя.
type Drama struct {
	id             int64
	userID         int64
	name           string
	year           int
	genre          string
	country        string
	doramatvRating float64   // рейтинг с doramatv.one
	watchStatus    WatchStatus
	currentEpisode int
	totalEpisodes  int
	doramatvURL    string
	tags           []string  // ["выходит", "переводится"] и т.п.
	createdAt      time.Time
	updatedAt      time.Time
}

// ReconstituteDrama восстанавливает Drama из БД без валидации.
func ReconstituteDrama(
	id, userID int64,
	name string,
	year int,
	genre, country string,
	doramatvRating float64,
	watchStatus WatchStatus,
	currentEpisode, totalEpisodes int,
	doramatvURL string,
	tags []string,
	createdAt, updatedAt time.Time,
) *Drama {
	return &Drama{
		id:             id,
		userID:         userID,
		name:           name,
		year:           year,
		genre:          genre,
		country:        country,
		doramatvRating: doramatvRating,
		watchStatus:    watchStatus,
		currentEpisode: currentEpisode,
		totalEpisodes:  totalEpisodes,
		doramatvURL:    doramatvURL,
		tags:           tags,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

// Геттеры
func (d *Drama) ID() int64              { return d.id }
func (d *Drama) UserID() int64          { return d.userID }
func (d *Drama) Name() string           { return d.name }
func (d *Drama) Year() int              { return d.year }
func (d *Drama) Genre() string          { return d.genre }
func (d *Drama) Country() string        { return d.country }
func (d *Drama) DoramatvRating() float64 { return d.doramatvRating }
func (d *Drama) WatchStatus() WatchStatus { return d.watchStatus }
func (d *Drama) CurrentEpisode() int    { return d.currentEpisode }
func (d *Drama) TotalEpisodes() int     { return d.totalEpisodes }
func (d *Drama) DoramatvURL() string    { return d.doramatvURL }
func (d *Drama) Tags() []string         { return d.tags }
func (d *Drama) CreatedAt() time.Time   { return d.createdAt }
func (d *Drama) UpdatedAt() time.Time   { return d.updatedAt }

// Badge — бэйдж, заработанный пользователем за активность.
type Badge struct {
	id          int64
	userID      int64
	code        string
	name        string
	description string
	icon        string
	earnedAt    time.Time
}

// ReconstituteBadge восстанавливает Badge из БД без валидации.
func ReconstituteBadge(id, userID int64, code, name, description, icon string, earnedAt time.Time) *Badge {
	return &Badge{
		id:          id,
		userID:      userID,
		code:        code,
		name:        name,
		description: description,
		icon:        icon,
		earnedAt:    earnedAt,
	}
}

// Геттеры
func (b *Badge) ID() int64          { return b.id }
func (b *Badge) UserID() int64      { return b.userID }
func (b *Badge) Code() string       { return b.code }
func (b *Badge) Name() string       { return b.name }
func (b *Badge) Description() string { return b.description }
func (b *Badge) Icon() string       { return b.icon }
func (b *Badge) EarnedAt() time.Time { return b.earnedAt }
