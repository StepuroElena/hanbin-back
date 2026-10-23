package streamingsite

import (
	"context"
	"errors"
	"testing"

	domain "github.com/hanbin/hanbin-back/internal/domain/streamingsite"
)

// fakeRepo — in-memory реализация domain.Repository для тестов сервиса.
type fakeRepo struct {
	nextID int64
	items  []*domain.StreamingSite
}

func (r *fakeRepo) Create(_ context.Context, s *domain.StreamingSite) (int64, error) {
	r.nextID++
	r.items = append(r.items, domain.Reconstitute(r.nextID, s.ProfileID(), s.Name(), s.URL(), s.Language(), s.Position(), s.Enabled(), s.CreatedAt(), s.UpdatedAt()))
	return r.nextID, nil
}

func (r *fakeRepo) GetAllByProfileID(_ context.Context, profileID int64) ([]*domain.StreamingSite, error) {
	var out []*domain.StreamingSite
	for _, s := range r.items {
		if s.ProfileID() == profileID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *fakeRepo) GetByID(_ context.Context, id int64) (*domain.StreamingSite, error) {
	for _, s := range r.items {
		if s.ID() == id {
			return domain.Reconstitute(s.ID(), s.ProfileID(), s.Name(), s.URL(), s.Language(), s.Position(), s.Enabled(), s.CreatedAt(), s.UpdatedAt()), nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeRepo) Update(_ context.Context, s *domain.StreamingSite) error {
	for i, x := range r.items {
		if x.ID() == s.ID() {
			r.items[i] = s
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

func TestDuplicateSitesRejectedCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	s := NewService(&fakeRepo{})

	netflix, err := s.Create(ctx, 1, CreateInput{Name: "Netflix", URL: "https://netflix.com"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	cases := []struct {
		in   CreateInput
		want error
	}{
		{CreateInput{Name: "  NETFLIX ", URL: "https://other.com"}, domain.ErrDuplicateName},
		{CreateInput{Name: "Нетфликс", URL: "http://www.Netflix.com/"}, domain.ErrDuplicateURL},
	}
	for _, c := range cases {
		if _, err := s.Create(ctx, 1, c.in); !errors.Is(err, c.want) {
			t.Fatalf("create %+v: want %v, got %v", c.in, c.want, err)
		}
	}

	// У другого профиля — можно.
	if _, err := s.Create(ctx, 2, CreateInput{Name: "netflix", URL: "netflix.com"}); err != nil {
		t.Fatalf("other profile: %v", err)
	}

	// Переименовать в занятое — нельзя, оставить своё (в другом регистре) — можно.
	viki, _ := s.Create(ctx, 1, CreateInput{Name: "Viki", URL: "https://viki.com"})
	if _, err := s.Update(ctx, 1, viki.ID, UpdateInput{Name: strPtr("netflix")}); !errors.Is(err, domain.ErrDuplicateName) {
		t.Fatalf("rename to taken: got %v", err)
	}
	if _, err := s.Update(ctx, 1, netflix.ID, UpdateInput{Name: strPtr("NetFlix")}); err != nil {
		t.Fatalf("rename self: %v", err)
	}
}
