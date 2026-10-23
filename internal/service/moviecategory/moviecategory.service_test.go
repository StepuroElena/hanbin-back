package moviecategory

import (
	"context"
	"errors"
	"testing"

	domain "github.com/hanbin/hanbin-back/internal/domain/moviecategory"
)

// fakeRepo — in-memory реализация domain.Repository для тестов сервиса.
type fakeRepo struct {
	nextID int64
	items  []*domain.MovieCategory
}

func (r *fakeRepo) Create(_ context.Context, c *domain.MovieCategory) (int64, error) {
	r.nextID++
	r.items = append(r.items, domain.Reconstitute(r.nextID, c.ProfileID(), c.Name(), c.Position(), c.Enabled(), c.CreatedAt(), c.UpdatedAt()))
	return r.nextID, nil
}

func (r *fakeRepo) GetAllByProfileID(_ context.Context, profileID int64) ([]*domain.MovieCategory, error) {
	var out []*domain.MovieCategory
	for _, c := range r.items {
		if c.ProfileID() == profileID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeRepo) GetByID(_ context.Context, id int64) (*domain.MovieCategory, error) {
	for _, c := range r.items {
		if c.ID() == id {
			return domain.Reconstitute(c.ID(), c.ProfileID(), c.Name(), c.Position(), c.Enabled(), c.CreatedAt(), c.UpdatedAt()), nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeRepo) Update(_ context.Context, c *domain.MovieCategory) error {
	for i, x := range r.items {
		if x.ID() == c.ID() {
			r.items[i] = c
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *fakeRepo) Delete(_ context.Context, id int64) error { return nil }

func (r *fakeRepo) CountByProfileID(_ context.Context, profileID int64) (int, error) {
	list, _ := r.GetAllByProfileID(context.Background(), profileID)
	return len(list), nil
}

func strPtr(s string) *string { return &s }

func TestDuplicateCategoriesRejectedCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	s := NewService(&fakeRepo{})

	evening, err := s.Create(ctx, 1, CreateInput{Name: "Для вечера"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Create(ctx, 1, CreateInput{Name: "  для   ВЕЧЕРА "}); !errors.Is(err, domain.ErrDuplicateName) {
		t.Fatalf("duplicate: got %v", err)
	}
	if _, err := s.Create(ctx, 2, CreateInput{Name: "для вечера"}); err != nil {
		t.Fatalf("other profile: %v", err)
	}

	other, _ := s.Create(ctx, 1, CreateInput{Name: "Атмосферное"})
	if _, err := s.Update(ctx, 1, other.ID, UpdateInput{Name: strPtr("ДЛЯ ВЕЧЕРА")}); !errors.Is(err, domain.ErrDuplicateName) {
		t.Fatalf("rename to taken: got %v", err)
	}
	if _, err := s.Update(ctx, 1, evening.ID, UpdateInput{Name: strPtr("для Вечера")}); err != nil {
		t.Fatalf("rename self: %v", err)
	}
}
