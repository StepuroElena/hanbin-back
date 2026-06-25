package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

// ── DTO для /users/me ─────────────────────────────────────────────────────────

// DramaOutput — дорама в ответе /users/me.
type DramaOutput struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	Year           int      `json:"year,omitempty"`
	Genre          string   `json:"genre,omitempty"`
	Country        string   `json:"country,omitempty"`
	Tags           []string `json:"tags"`
	DoramatvRating float64  `json:"doramatv_rating,omitempty"`
	WatchStatus    string   `json:"watch_status"`
	CurrentEpisode int      `json:"current_episode"`
	TotalEpisodes  int      `json:"total_episodes,omitempty"`
	DoramatvURL    string   `json:"doramatv_url,omitempty"`
}

// BadgeOutput — бэйдж в ответе /users/me.
type BadgeOutput struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	EarnedAt    string `json:"earned_at"`
}

// MeOutput — полный ответ эндпоинта GET /api/v1/users/me.
type MeOutput struct {
	UserID int64         `json:"user_id"`
	Name   string        `json:"name"`
	Email  string        `json:"email"`
	Dramas []DramaOutput `json:"dramas"`
	Badges []BadgeOutput `json:"badges"`
}

// GetMe возвращает данные текущего пользователя по JWT-токену.
// token — значение из заголовка Authorization: Bearer <token>.
func (s *Service) GetMe(ctx context.Context, token string) (*MeOutput, error) {
	userID, err := parseUserIDFromToken(token)
	if err != nil {
		return nil, fmt.Errorf("service.GetMe: %w", domain.ErrInvalidCredentials)
	}

	// Загружаем пользователя.
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service.GetMe: %w", err)
	}

	// Загружаем дорамы.
	dramas, err := s.dramaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service.GetMe: dramas: %w", err)
	}

	// Загружаем бэйджи.
	badges, err := s.badgeRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service.GetMe: badges: %w", err)
	}

	return &MeOutput{
		UserID: u.ID(),
		Name:   u.Name(),
		Email:  u.Email(),
		Dramas: toDramaOutputs(dramas),
		Badges: toBadgeOutputs(badges),
	}, nil
}

// ── Вспомогательные функции ───────────────────────────────────────────────────

// parseUserIDFromToken валидирует JWT и возвращает userID из subject.
func parseUserIDFromToken(raw string) (int64, error) {
	raw = strings.TrimPrefix(raw, "Bearer ")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty token")
	}

	secret := jwtSecret()
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !tok.Valid {
		return 0, fmt.Errorf("invalid token: %w", err)
	}

	sub, err := tok.Claims.GetSubject()
	if err != nil || sub == "" {
		return 0, fmt.Errorf("missing subject in token")
	}

	id, err := strconv.ParseInt(sub, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid subject in token")
	}
	return id, nil
}

func toDramaOutputs(dramas []*domain.Drama) []DramaOutput {
	out := make([]DramaOutput, 0, len(dramas))
	for _, d := range dramas {
		tags := d.Tags()
		if tags == nil {
			tags = []string{}
		}
		out = append(out, DramaOutput{
			ID:             d.ID(),
			Name:           d.Name(),
			Year:           d.Year(),
			Genre:          d.Genre(),
			Country:        d.Country(),
			Tags:           tags,
			DoramatvRating: d.DoramatvRating(),
			WatchStatus:    string(d.WatchStatus()),
			CurrentEpisode: d.CurrentEpisode(),
			TotalEpisodes:  d.TotalEpisodes(),
			DoramatvURL:    d.DoramatvURL(),
		})
	}
	return out
}

func toBadgeOutputs(badges []*domain.Badge) []BadgeOutput {
	out := make([]BadgeOutput, 0, len(badges))
	for _, b := range badges {
		out = append(out, BadgeOutput{
			Code:        b.Code(),
			Name:        b.Name(),
			Description: b.Description(),
			Icon:        b.Icon(),
			EarnedAt:    b.EarnedAt().Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}
