package voiceover

import (
	"context"
	"fmt"
	"strings"

	domain "github.com/hanbin/hanbin-back/internal/domain/voiceover"
)

// Service реализует use-case'ы для работы с озвучками.
//
// Правила:
//   - дефолтные озвучки (общие для всех, лежат в БД с profile_id NULL) можно только выключить/включить;
//   - свои озвучки можно добавлять, переименовывать, удалять и выключать;
//   - названия уникальны в пределах списка пользователя (дефолтные + свои) без учёта регистра и пунктуации.
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// ── DTO ───────────────────────────────────────────────────────────────────────

// Output — публичное представление озвучки.
type Output struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	IsDefault bool   `json:"is_default"`
}

// CreateInput — тело запроса на добавление своей озвучки.
type CreateInput struct {
	Name string `json:"name"`
}

// UpdateInput — частичное обновление. name доступно только для своих озвучек.
type UpdateInput struct {
	Name    *string `json:"name"`
	Enabled *bool   `json:"enabled"`
}

// MatchOutput — результат сопоставления строки со скрейпа со списком пользователя.
// Name пустой — совпадений нет.
type MatchOutput struct {
	Name string `json:"name"`
}

// ── Use cases ─────────────────────────────────────────────────────────────────

// List возвращает полный список озвучек профиля (дефолтные + свои) с тоглами — для Настроек и дропдаунов.
func (s *Service) List(ctx context.Context, profileID int64) ([]Output, error) {
	list, err := s.repo.ListForProfile(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.List: %w", err)
	}
	return toOutputs(list), nil
}

// ListDefaults возвращает дефолтный набор — для гостя без аккаунта.
func (s *Service) ListDefaults(ctx context.Context) ([]Output, error) {
	list, err := s.repo.ListDefaults(ctx)
	if err != nil {
		return nil, fmt.Errorf("service.ListDefaults: %w", err)
	}
	return toOutputs(list), nil
}

// Create добавляет пользователю свою озвучку. Включена по умолчанию.
func (s *Service) Create(ctx context.Context, profileID int64, in CreateInput) (*Output, error) {
	pos, err := s.repo.NextCustomPosition(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	v, err := domain.NewCustom(profileID, in.Name, pos)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}
	if err := s.ensureUniqueName(ctx, profileID, v.Name(), 0); err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	id, err := s.repo.Create(ctx, v)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	out := toOutput(domain.Reconstitute(id, profileID, v.Name(), v.Position(), v.Enabled(), v.CreatedAt(), v.UpdatedAt()))
	return &out, nil
}

// Update переименовывает свою озвучку и/или переключает тогл (для любой доступной озвучки).
func (s *Service) Update(ctx context.Context, profileID, id int64, in UpdateInput) (*Output, error) {
	v, err := s.repo.GetForProfile(ctx, profileID, id)
	if err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}

	if in.Name != nil {
		oldName := v.Name()
		if err := v.SetName(*in.Name); err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
		if v.Name() != oldName {
			if err := s.ensureUniqueName(ctx, profileID, v.Name(), v.ID()); err != nil {
				return nil, fmt.Errorf("service.Update: %w", err)
			}
			if err := s.repo.Rename(ctx, profileID, v.ID(), oldName, v.Name()); err != nil {
				return nil, fmt.Errorf("service.Update: %w", err)
			}
		}
	}

	if in.Enabled != nil {
		v.SetEnabled(*in.Enabled)
		if err := s.repo.SetEnabled(ctx, profileID, v.ID(), v.Enabled()); err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
	}

	out := toOutput(v)
	return &out, nil
}

// Delete удаляет свою озвучку. Дефолтные удалять нельзя — только выключить.
// У дорам, где она уже выбрана, название остаётся как есть.
func (s *Service) Delete(ctx context.Context, profileID, id int64) error {
	v, err := s.repo.GetForProfile(ctx, profileID, id)
	if err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	if v.IsDefault() {
		return fmt.Errorf("service.Delete: %w", domain.ErrDefaultReadOnly)
	}
	if err := s.repo.DeleteCustom(ctx, profileID, id); err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	return nil
}

// Match сопоставляет строку озвучки со скрейпа (может быть несколько студий через запятую,
// с другим написанием, напр. "DubLikTV, Light Breeze") с включёнными озвучками пользователя.
// Возвращает первое совпадение по порядку студий в строке. profileID == 0 (гость) — по дефолтным.
func (s *Service) Match(ctx context.Context, profileID int64, raw string) (*MatchOutput, error) {
	var (
		list []*domain.Voiceover
		err  error
	)
	if profileID > 0 {
		list, err = s.repo.ListForProfile(ctx, profileID)
	} else {
		list, err = s.repo.ListDefaults(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("service.Match: %w", err)
	}
	return &MatchOutput{Name: matchName(list, raw)}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func matchName(list []*domain.Voiceover, raw string) string {
	byNorm := make(map[string]string, len(list))
	for _, v := range list {
		if !v.Enabled() {
			continue
		}
		if key := domain.NormalizeName(v.Name()); key != "" {
			if _, exists := byNorm[key]; !exists {
				byNorm[key] = v.Name()
			}
		}
	}
	for _, candidate := range strings.Split(raw, ",") {
		if name, ok := byNorm[domain.NormalizeName(candidate)]; ok {
			return name
		}
	}
	return ""
}

// ensureUniqueName проверяет, что в списке профиля (дефолтные + свои) нет озвучки с таким же
// нормализованным названием. excludeID — сама переименовываемая озвучка.
func (s *Service) ensureUniqueName(ctx context.Context, profileID int64, name string, excludeID int64) error {
	list, err := s.repo.ListForProfile(ctx, profileID)
	if err != nil {
		return err
	}
	key := domain.NormalizeName(name)
	for _, v := range list {
		if v.ID() == excludeID {
			continue
		}
		if (key != "" && domain.NormalizeName(v.Name()) == key) || strings.EqualFold(v.Name(), name) {
			return domain.ErrDuplicateName
		}
	}
	return nil
}

func toOutputs(list []*domain.Voiceover) []Output {
	out := make([]Output, 0, len(list))
	for _, v := range list {
		out = append(out, toOutput(v))
	}
	return out
}

func toOutput(v *domain.Voiceover) Output {
	return Output{
		ID:        v.ID(),
		Name:      v.Name(),
		Enabled:   v.Enabled(),
		IsDefault: v.IsDefault(),
	}
}
