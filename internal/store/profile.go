package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// UserProfile is the optional, public part of an account. Callers validate the
// fields; the store only persists them.
type UserProfile struct {
	Bio         string     `json:"bio"`
	Pronouns    string     `json:"pronouns"`
	Website     string     `json:"website"`
	Timezone    string     `json:"timezone"`
	SocialLinks []string   `json:"social_links"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// GetUserProfile returns the user's profile, or an empty one if it was never saved.
func (s *Store) GetUserProfile(ctx context.Context, userID int64) (UserProfile, error) {
	var (
		profile UserProfile
		links   []byte
		updated time.Time
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT bio, pronouns, website, timezone, social_links, updated_at
		FROM user_profiles
		WHERE user_id = $1
	`, userID).Scan(&profile.Bio, &profile.Pronouns, &profile.Website, &profile.Timezone, &links, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return UserProfile{SocialLinks: []string{}}, nil
	}
	if err != nil {
		return UserProfile{}, err
	}
	if err := json.Unmarshal(links, &profile.SocialLinks); err != nil {
		return UserProfile{}, err
	}
	if profile.SocialLinks == nil {
		profile.SocialLinks = []string{}
	}
	profile.UpdatedAt = &updated
	return profile, nil
}

// SaveUserProfile creates or replaces the user's profile.
func (s *Store) SaveUserProfile(ctx context.Context, userID int64, profile UserProfile) (UserProfile, error) {
	if profile.SocialLinks == nil {
		profile.SocialLinks = []string{}
	}
	links, err := json.Marshal(profile.SocialLinks)
	if err != nil {
		return UserProfile{}, err
	}
	var updated time.Time
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO user_profiles (user_id, bio, pronouns, website, timezone, social_links, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			bio = EXCLUDED.bio,
			pronouns = EXCLUDED.pronouns,
			website = EXCLUDED.website,
			timezone = EXCLUDED.timezone,
			social_links = EXCLUDED.social_links,
			updated_at = NOW()
		RETURNING updated_at
	`, userID, profile.Bio, profile.Pronouns, profile.Website, profile.Timezone, string(links)).Scan(&updated)
	if err != nil {
		if isForeignKeyViolation(err) {
			return UserProfile{}, ErrUserNotFound
		}
		return UserProfile{}, err
	}
	profile.UpdatedAt = &updated
	return profile, nil
}

// isForeignKeyViolation reports whether err is Postgres error 23503, a missing referenced row.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
