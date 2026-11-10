package store

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
)

const userCols = `id, email, coalesce(password_hash,''), first_name, last_name, phone, role, tier, points,
	marketing_email, marketing_sms, marketing_whatsapp, email_verified_at, created_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Phone, &u.Role, &u.Tier, &u.Points,
		&u.MarketingEmail, &u.MarketingSMS, &u.MarketingWhatsApp, &u.EmailVerifiedAt, &u.CreatedAt)
	if err != nil {
		return nil, notFound(err)
	}
	return &u, nil
}

func (s *Store) CreateUser(ctx context.Context, email, hash, first, last string, marketing bool) (*User, error) {
	row := s.db.QueryRow(ctx, `insert into users (email, password_hash, first_name, last_name, marketing_email)
		values ($1,$2,$3,$4,$5) returning `+userCols, strings.TrimSpace(email), hash, first, last, marketing)
	u, err := scanUser(row)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_norm_idx") {
			return nil, httpx.E(409, "email_taken", "This email is already registered. Try signing in instead.")
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(s.db.QueryRow(ctx, `select `+userCols+` from users where email_norm = lower($1)`, strings.TrimSpace(email)))
}

func (s *Store) UserByID(ctx context.Context, id string) (*User, error) {
	return scanUser(s.db.QueryRow(ctx, `select `+userCols+` from users where id = $1`, id))
}

func (s *Store) TouchLogin(ctx context.Context, id string) {
	_, _ = s.db.Exec(ctx, `update users set last_login_at = now() where id = $1`, id)
}

var userUpdatable = map[string]bool{"first_name": true, "last_name": true, "phone": true,
	"marketing_email": true, "marketing_sms": true, "marketing_whatsapp": true}

func (s *Store) UpdateUser(ctx context.Context, id string, cols map[string]any) (*User, error) {
	set, args, err := buildUpdate(cols, userUpdatable, 2)
	if err != nil {
		return nil, err
	}
	return scanUser(s.db.QueryRow(ctx, `update users set `+set+` where id = $1 returning `+userCols, append([]any{id}, args...)...))
}

func (s *Store) SetPassword(ctx context.Context, id, hash string) error {
	_, err := s.db.Exec(ctx, `update users set password_hash = $2 where id = $1`, id, hash)
	return err
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	return s.Tx(ctx, func(tx pgx.Tx) error {
		// Keep orders for accounting but anonymise them (UK GDPR erasure).
		if _, err := tx.Exec(ctx, `update orders set user_id = null, email = 'deleted@anonymised.invalid',
			shipping_address = jsonb_build_object('firstName','Deleted','lastName','Customer','line1','—','city','—','postcode','—','country','GB')
			where user_id = $1`, id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `delete from users where id = $1`, id)
		return err
	})
}

// ── Refresh tokens ───────────────────────────────────────────────────────────

func (s *Store) SaveRefresh(ctx context.Context, userID, hash, ua string, ttl time.Duration) error {
	_, err := s.db.Exec(ctx, `insert into refresh_tokens (user_id, token_hash, user_agent, expires_at) values ($1,$2,$3,$4)`,
		userID, hash, ua, time.Now().Add(ttl))
	return err
}

// ConsumeRefresh revokes the token and returns its user. Rotating tokens means
// a stolen token is only valid until the legitimate client refreshes.
func (s *Store) ConsumeRefresh(ctx context.Context, hash string) (string, error) {
	var userID string
	err := s.db.QueryRow(ctx, `update refresh_tokens set revoked_at = now()
		where token_hash = $1 and revoked_at is null and expires_at > now() returning user_id`, hash).Scan(&userID)
	if err != nil {
		return "", httpx.E(401, "invalid_refresh", "Your session has expired. Please sign in again.")
	}
	return userID, nil
}

func (s *Store) RevokeAllRefresh(ctx context.Context, userID string) error {
	_, err := s.db.Exec(ctx, `update refresh_tokens set revoked_at = now() where user_id = $1 and revoked_at is null`, userID)
	return err
}

// ── Password resets ──────────────────────────────────────────────────────────

func (s *Store) SaveReset(ctx context.Context, userID, hash string) error {
	_, err := s.db.Exec(ctx, `insert into password_resets (user_id, token_hash, expires_at) values ($1,$2,now() + interval '1 hour')`, userID, hash)
	return err
}

func (s *Store) ConsumeReset(ctx context.Context, hash string) (string, error) {
	var userID string
	err := s.db.QueryRow(ctx, `update password_resets set used_at = now()
		where token_hash = $1 and used_at is null and expires_at > now() returning user_id`, hash).Scan(&userID)
	if err != nil {
		return "", httpx.E(401, "invalid_reset", "This link is invalid or has expired.")
	}
	return userID, nil
}

// ── Addresses ────────────────────────────────────────────────────────────────

const addrCols = `id, label, first_name, last_name, line1, line2, city, postcode, country, phone, is_default`

func scanAddr(rows pgx.Rows) (Address, error) {
	var a Address
	err := rows.Scan(&a.ID, &a.Label, &a.FirstName, &a.LastName, &a.Line1, &a.Line2, &a.City, &a.Postcode, &a.Country, &a.Phone, &a.IsDefault)
	return a, err
}

func (s *Store) ListAddresses(ctx context.Context, userID string) ([]Address, error) {
	rows, err := s.db.Query(ctx, `select `+addrCols+` from addresses where user_id = $1 order by is_default desc, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Address{}
	for rows.Next() {
		a, err := scanAddr(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) UpsertAddress(ctx context.Context, userID string, a Address) (Address, error) {
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		if a.IsDefault {
			if _, err := tx.Exec(ctx, `update addresses set is_default = false where user_id = $1`, userID); err != nil {
				return err
			}
		}
		if a.ID == 0 {
			return tx.QueryRow(ctx, `insert into addresses (user_id,label,first_name,last_name,line1,line2,city,postcode,country,phone,is_default)
				values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) returning id`,
				userID, a.Label, a.FirstName, a.LastName, a.Line1, a.Line2, a.City, a.Postcode, a.Country, a.Phone, a.IsDefault).Scan(&a.ID)
		}
		tag, err := tx.Exec(ctx, `update addresses set label=$3,first_name=$4,last_name=$5,line1=$6,line2=$7,city=$8,postcode=$9,country=$10,phone=$11,is_default=$12
			where id = $1 and user_id = $2`,
			a.ID, userID, a.Label, a.FirstName, a.LastName, a.Line1, a.Line2, a.City, a.Postcode, a.Country, a.Phone, a.IsDefault)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return httpx.ErrNotFound
		}
		return nil
	})
	return a, err
}

func (s *Store) DeleteAddress(ctx context.Context, userID string, id int64) error {
	tag, err := s.db.Exec(ctx, `delete from addresses where id = $1 and user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

// ── Scent profile / wishlist / recently viewed ───────────────────────────────

func (s *Store) ScentProfile(ctx context.Context, userID string) (*ScentProfile, error) {
	var p ScentProfile
	err := s.db.QueryRow(ctx, `select answers, families, liked_notes, avoided_notes, updated_at from scent_profiles where user_id = $1`, userID).
		Scan(&p.Answers, &p.Families, &p.LikedNotes, &p.AvoidedNotes, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *Store) SaveScentProfile(ctx context.Context, userID string, p ScentProfile) error {
	if p.Answers == nil {
		p.Answers = []byte("{}")
	}
	_, err := s.db.Exec(ctx, `insert into scent_profiles (user_id, answers, families, liked_notes, avoided_notes)
		values ($1,$2,$3,$4,$5) on conflict (user_id) do update set answers = excluded.answers, families = excluded.families,
		liked_notes = excluded.liked_notes, avoided_notes = excluded.avoided_notes, updated_at = now()`,
		userID, p.Answers, p.Families, p.LikedNotes, p.AvoidedNotes)
	return err
}

func (s *Store) Wishlist(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `select p.slug from wishlist w join products p on p.id = w.product_id where w.user_id = $1 order by w.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		out = append(out, slug)
	}
	return out, rows.Err()
}

func (s *Store) ToggleWishlist(ctx context.Context, userID, slug string) (bool, error) {
	var pid int64
	if err := s.db.QueryRow(ctx, `select id from products where slug = $1`, slug).Scan(&pid); err != nil {
		return false, notFound(err)
	}
	tag, err := s.db.Exec(ctx, `delete from wishlist where user_id = $1 and product_id = $2`, userID, pid)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() > 0 {
		return false, nil
	}
	_, err = s.db.Exec(ctx, `insert into wishlist (user_id, product_id) values ($1,$2) on conflict do nothing`, userID, pid)
	return true, err
}

func (s *Store) RecordView(ctx context.Context, userID string, productID int64) {
	_, _ = s.db.Exec(ctx, `insert into recently_viewed (user_id, product_id) values ($1,$2)
		on conflict (user_id, product_id) do update set viewed_at = now()`, userID, productID)
}

// ExportUserData returns everything we hold about a user (UK GDPR access request).
func (s *Store) ExportUserData(ctx context.Context, userID string) (map[string]any, error) {
	u, err := s.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	addrs, _ := s.ListAddresses(ctx, userID)
	orders, _ := s.ListOrdersForUser(ctx, userID)
	wish, _ := s.Wishlist(ctx, userID)
	profile, _ := s.ScentProfile(ctx, userID)
	return map[string]any{
		"profile": u, "addresses": addrs, "orders": orders, "wishlist": wish, "scentProfile": profile,
		"exportedAt": time.Now().UTC(),
	}, nil
}
