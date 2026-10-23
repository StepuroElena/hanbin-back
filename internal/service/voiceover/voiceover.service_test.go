package voiceover

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/voiceover"
)

// fakeRepo — in-memory реализация domain.Repository для тестов сервиса.
type fakeRepo struct {
	nextID   int64
	items    map[int64]*domain.Voiceover // profileID 0 = дефолтная
	settings map[[2]int64]bool           // [profileID, voiceoverID] → enabled
	renamed  [][2]string                 // журнал переименований дорам (old, new)
}

func newFakeRepo(defaults ...string) *fakeRepo {
	r := &fakeRepo{items: map[int64]*domain.Voiceover{}, settings: map[[2]int64]bool{}}
	for i, name := range defaults {
		r.nextID++
		r.items[r.nextID] = domain.Reconstitute(r.nextID, 0, name, i, true, time.Time{}, time.Time{})
	}
	return r
}

func (r *fakeRepo) view(profileID int64, v *domain.Voiceover) *domain.Voiceover {
	enabled := true
	if e, ok := r.settings[[2]int64{profileID, v.ID()}]; ok {
		enabled = e
	}
	return domain.Reconstitute(v.ID(), v.ProfileID(), v.Name(), v.Position(), enabled, v.CreatedAt(), v.UpdatedAt())
}

func (r *fakeRepo) ListForProfile(_ context.Context, profileID int64) ([]*domain.Voiceover, error) {
	var out []*domain.Voiceover
	for id := int64(1); id <= r.nextID; id++ {
		if v, ok := r.items[id]; ok && (v.IsDefault() || v.ProfileID() == profileID) {
			out = append(out, r.view(profileID, v))
		}
	}
	return out, nil
}

func (r *fakeRepo) ListDefaults(_ context.Context) ([]*domain.Voiceover, error) {
	var out []*domain.Voiceover
	for id := int64(1); id <= r.nextID; id++ {
		if v, ok := r.items[id]; ok && v.IsDefault() {
			out = append(out, v)
		}
	}
	return out, nil
}

func (r *fakeRepo) GetForProfile(_ context.Context, profileID, id int64) (*domain.Voiceover, error) {
	v, ok := r.items[id]
	if !ok || (!v.IsDefault() && v.ProfileID() != profileID) {
		return nil, domain.ErrNotFound
	}
	return r.view(profileID, v), nil
}

func (r *fakeRepo) NextCustomPosition(_ context.Context, _ int64) (int, error) { return 0, nil }

func (r *fakeRepo) Create(_ context.Context, v *domain.Voiceover) (int64, error) {
	r.nextID++
	r.items[r.nextID] = domain.Reconstitute(r.nextID, v.ProfileID(), v.Name(), v.Position(), true, time.Time{}, time.Time{})
	return r.nextID, nil
}

func (r *fakeRepo) Rename(_ context.Context, profileID, id int64, oldName, newName string) error {
	v := r.items[id]
	r.items[id] = domain.Reconstitute(id, profileID, newName, v.Position(), true, time.Time{}, time.Time{})
	r.renamed = append(r.renamed, [2]string{oldName, newName})
	return nil
}

func (r *fakeRepo) SetEnabled(_ context.Context, profileID, id int64, enabled bool) error {
	r.settings[[2]int64{profileID, id}] = enabled
	return nil
}

func (r *fakeRepo) DeleteCustom(_ context.Context, profileID, id int64) error {
	if v, ok := r.items[id]; !ok || v.ProfileID() != profileID {
		return domain.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

const (
	alice int64 = 10
	bob   int64 = 20
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestDefaultsAreToggleOnlyAndPerUser(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo("Light Breeze", "DubLik.TV")
	s := NewService(repo)

	// Тогл дефолтной — можно, и только у себя.
	out, err := s.Update(ctx, alice, 1, UpdateInput{Enabled: boolPtr(false)})
	if err != nil || out.Enabled || !out.IsDefault {
		t.Fatalf("toggle default: out=%+v err=%v", out, err)
	}
	bobList, _ := s.List(ctx, bob)
	if !bobList[0].Enabled {
		t.Fatal("toggle of alice must not affect bob")
	}

	// Переименовать или удалить дефолтную — нельзя.
	if _, err := s.Update(ctx, alice, 1, UpdateInput{Name: strPtr("X")}); !errors.Is(err, domain.ErrDefaultReadOnly) {
		t.Fatalf("rename default: want ErrDefaultReadOnly, got %v", err)
	}
	if err := s.Delete(ctx, alice, 1); !errors.Is(err, domain.ErrDefaultReadOnly) {
		t.Fatalf("delete default: want ErrDefaultReadOnly, got %v", err)
	}
}

func TestCustomCRUDAndIsolation(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo("Light Breeze")
	s := NewService(repo)

	created, err := s.Create(ctx, alice, CreateInput{Name: "  LostFilm  "})
	if err != nil || created.Name != "LostFilm" || created.IsDefault || !created.Enabled {
		t.Fatalf("create: out=%+v err=%v", created, err)
	}

	// Дубли: с дефолтной (другое написание) и со своей.
	if _, err := s.Create(ctx, alice, CreateInput{Name: "light-breeze"}); !errors.Is(err, domain.ErrDuplicateName) {
		t.Fatalf("dup with default: got %v", err)
	}
	if _, err := s.Create(ctx, alice, CreateInput{Name: "lostfilm"}); !errors.Is(err, domain.ErrDuplicateName) {
		t.Fatalf("dup with own: got %v", err)
	}
	// У другого пользователя такое же название — можно.
	if _, err := s.Create(ctx, bob, CreateInput{Name: "LostFilm"}); err != nil {
		t.Fatalf("same name for bob: %v", err)
	}
	if _, err := s.Create(ctx, alice, CreateInput{Name: "   "}); !errors.Is(err, domain.ErrNameRequired) {
		t.Fatalf("empty name: got %v", err)
	}

	// Переименование своей — переносится на дорамы.
	renamed, err := s.Update(ctx, alice, created.ID, UpdateInput{Name: strPtr("LostFilm HD")})
	if err != nil || renamed.Name != "LostFilm HD" {
		t.Fatalf("rename: out=%+v err=%v", renamed, err)
	}
	if len(repo.renamed) != 1 || repo.renamed[0] != [2]string{"LostFilm", "LostFilm HD"} {
		t.Fatalf("dramas rename not propagated: %v", repo.renamed)
	}

	// Чужую нельзя ни менять, ни удалять.
	if _, err := s.Update(ctx, bob, created.ID, UpdateInput{Enabled: boolPtr(false)}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("bob edits alice: got %v", err)
	}
	if err := s.Delete(ctx, bob, created.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("bob deletes alice: got %v", err)
	}

	if err := s.Delete(ctx, alice, created.ID); err != nil {
		t.Fatalf("delete own: %v", err)
	}
	list, _ := s.List(ctx, alice)
	if len(list) != 1 || !list[0].IsDefault {
		t.Fatalf("after delete: %+v", list)
	}
}

func TestMatch(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo("Light Breeze", "DubLik.TV", "SoftBox")
	s := NewService(repo)

	got, _ := s.Match(ctx, alice, "DubLikTV, Light Breeze, Русские субтитры")
	if got.Name != "DubLik.TV" {
		t.Fatalf("match first studio: got %q", got.Name)
	}

	// Выключенная озвучка не сопоставляется — берём следующую из строки.
	_, _ = s.Update(ctx, alice, 2, UpdateInput{Enabled: boolPtr(false)})
	got, _ = s.Match(ctx, alice, "DubLikTV, Light Breeze")
	if got.Name != "Light Breeze" {
		t.Fatalf("disabled skipped: got %q", got.Name)
	}

	got, _ = s.Match(ctx, alice, "Неизвестная студия")
	if got.Name != "" {
		t.Fatalf("no match: got %q", got.Name)
	}
}

func TestMatchGuestUsesDefaults(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo("Light Breeze")
	s := NewService(repo)
	_, _ = s.Create(ctx, alice, CreateInput{Name: "LostFilm"})

	if got, _ := s.Match(ctx, 0, "LostFilm"); got.Name != "" {
		t.Fatalf("guest must not see custom voiceovers: %q", got.Name)
	}
	if got, _ := s.Match(ctx, 0, "light breeze"); got.Name != "Light Breeze" {
		t.Fatalf("guest default match: %q", got.Name)
	}
}
