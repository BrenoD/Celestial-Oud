package store

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/mail"
)

// ── Reviews ──────────────────────────────────────────────────────────────────

const reviewCols = `r.id, r.product_id, p.name, p.slug, r.author_name, r.rating, r.title, r.body, r.longevity, r.verified, r.approved, r.created_at`

func scanReview(rows pgx.Rows) (Review, error) {
	var r Review
	err := rows.Scan(&r.ID, &r.ProductID, &r.ProductName, &r.ProductSlug, &r.Author, &r.Rating, &r.Title, &r.Body, &r.Longevity, &r.Verified, &r.Approved, &r.CreatedAt)
	return r, err
}

func (s *Store) ProductReviews(ctx context.Context, slug string) ([]Review, error) {
	return s.queryReviews(ctx, `where p.slug = $1 and r.approved order by r.created_at desc limit 50`, slug)
}

func (s *Store) AllReviews(ctx context.Context, pending bool) ([]Review, error) {
	w := ``
	if pending {
		w = `where not r.approved`
	}
	return s.queryReviews(ctx, w+` order by r.created_at desc limit 500`)
}

func (s *Store) queryReviews(ctx context.Context, tail string, args ...any) ([]Review, error) {
	rows, err := s.db.Query(ctx, `select `+reviewCols+` from product_reviews r join products p on p.id = r.product_id `+tail, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Review{}
	for rows.Next() {
		r, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type ReviewInput struct {
	Rating    int     `json:"rating"`
	Title     *string `json:"title"`
	Body      *string `json:"body"`
	Longevity *string `json:"longevity"`
	Author    string  `json:"author"`
}

func (s *Store) SubmitReview(ctx context.Context, slug string, userID *string, in ReviewInput) (*Review, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return nil, httpx.BadRequest("The rating must be between 1 and 5.")
	}
	in.Author = strings.TrimSpace(in.Author)
	if in.Author == "" {
		return nil, httpx.BadRequest("A name is required.")
	}
	if err := httpx.CheckLengths(httpx.Limit{Name: "Name", Value: in.Author, Max: 60}, httpx.Limit{Name: "Title", Value: deref(in.Title), Max: 120},
		httpx.Limit{Name: "Review", Value: deref(in.Body), Max: 2000}, httpx.Limit{Name: "Longevity", Value: deref(in.Longevity), Max: 40}); err != nil {
		return nil, err
	}
	var pid int64
	if err := s.db.QueryRow(ctx, `select id from products where slug = $1`, slug).Scan(&pid); err != nil {
		return nil, notFound(err)
	}
	verified := false
	var orderID *string
	if userID != nil {
		_ = s.db.QueryRow(ctx, `select o.id from orders o join order_items i on i.order_id = o.id
			where o.user_id = $1 and i.product_id = $2 and o.status in ('paid','processing','dispatched','delivered') limit 1`, *userID, pid).Scan(&orderID)
		verified = orderID != nil
	}
	var id int64
	err := s.db.QueryRow(ctx, `insert into product_reviews (product_id, user_id, order_id, author_name, rating, title, body, longevity, verified)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9) returning id`, pid, userID, orderID, in.Author, in.Rating, in.Title, in.Body, in.Longevity, verified).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "product_reviews_one_per_user") {
			return nil, httpx.E(409, "already_reviewed", "You have already reviewed this product.")
		}
		return nil, err
	}
	rs, err := s.queryReviews(ctx, `where r.id = $1`, id)
	if err != nil || len(rs) == 0 {
		return nil, err
	}
	return &rs[0], nil
}

var reviewUpdatable = map[string]bool{"title": true, "body": true, "rating": true, "approved": true, "longevity": true}

func (s *Store) UpdateReview(ctx context.Context, id int64, cols map[string]any) error {
	set, args, err := buildUpdate(cols, reviewUpdatable, 2)
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `update product_reviews set `+set+` where id = $1`, append([]any{id}, args...)...)
	if err == nil && tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return err
}

func (s *Store) DeleteReview(ctx context.Context, id int64) error {
	_, err := s.db.Exec(ctx, `delete from product_reviews where id = $1`, id)
	return err
}

// ── Newsletter & stock alerts ────────────────────────────────────────────────

// Subscribe records a pending (unconfirmed) subscription with the hash of a
// confirmation token. It reports whether a confirmation email is needed
// (false for an already-confirmed, active subscriber). Double opt-in keeps the
// list PECR-compliant and stops anyone signing up someone else.
func (s *Store) Subscribe(ctx context.Context, email, name, source, tokenHash string) (bool, error) {
	var confirmed, unsub *time.Time
	err := s.db.QueryRow(ctx, `insert into newsletter_subs (email, name, source, confirm_token_hash, unsubscribe_token)
		values ($1, nullif($2,''), nullif($3,''), $4, replace(gen_random_uuid()::text,'-','') || replace(gen_random_uuid()::text,'-',''))
		on conflict (email) do update set
		  name = coalesce(excluded.name, newsletter_subs.name),
		  confirm_token_hash = case when newsletter_subs.confirmed_at is null or newsletter_subs.unsubscribed_at is not null
		                            then excluded.confirm_token_hash else newsletter_subs.confirm_token_hash end
		returning confirmed_at, unsubscribed_at`, email, strings.TrimSpace(name), source, tokenHash).Scan(&confirmed, &unsub)
	if err != nil {
		return false, err
	}
	return confirmed == nil || unsub != nil, nil
}

// ConfirmSubscription activates the subscription for a confirmation token and
// returns the email, name and whether this was the first confirmation.
func (s *Store) ConfirmSubscription(ctx context.Context, tokenHash string) (string, string, bool, error) {
	var email string
	var name *string
	var was bool
	err := s.db.QueryRow(ctx, `update newsletter_subs n set confirmed_at = coalesce(n.confirmed_at, now()), unsubscribed_at = null, confirm_token_hash = null
		from (select email, confirmed_at is not null as was from newsletter_subs where confirm_token_hash = $1) o
		where n.email = o.email returning n.email, n.name, o.was`, tokenHash).Scan(&email, &name, &was)
	if err != nil {
		return "", "", false, httpx.E(404, "invalid_token", "This confirmation link is invalid or has already been used.")
	}
	return email, deref(name), !was, nil
}

// Unsubscribe uses the per-subscriber token from the email footer.
func (s *Store) Unsubscribe(ctx context.Context, token string) error {
	if len(token) < 32 {
		return httpx.E(404, "invalid_token", "This unsubscribe link is invalid.")
	}
	tag, err := s.db.Exec(ctx, `update newsletter_subs set unsubscribed_at = coalesce(unsubscribed_at, now()) where unsubscribe_token = $1`, token)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.E(404, "invalid_token", "This unsubscribe link is invalid.")
	}
	return nil
}

// ActiveSubscribers returns confirmed, still-subscribed recipients among
// emails, mapped to their unsubscribe tokens.
func (s *Store) ActiveSubscribers(ctx context.Context, emails []string) (map[string]string, error) {
	rows, err := s.db.Query(ctx, `select email, unsubscribe_token from newsletter_subs
		where confirmed_at is not null and unsubscribed_at is null and email = any($1)`, emails)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var e, t string
		if err := rows.Scan(&e, &t); err != nil {
			return nil, err
		}
		out[e] = t
	}
	return out, rows.Err()
}

func (s *Store) ListSubscribers(ctx context.Context) ([]NewsletterSub, error) {
	rows, err := s.db.Query(ctx, `select n.email, n.name, n.source, n.confirmed_at, n.unsubscribed_at, n.created_at,
		(select count(*) from orders o where lower(o.email) = n.email and o.status <> 'cancelled'),
		coalesce((select u.tier from users u where u.email_norm = n.email), 'Bronze')
		from newsletter_subs n order by n.created_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []NewsletterSub{}
	for rows.Next() {
		var n NewsletterSub
		if err := rows.Scan(&n.Email, &n.Name, &n.Source, &n.ConfirmedAt, &n.UnsubscribedAt, &n.CreatedAt, &n.OrderCount, &n.Tier); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) NotifyStock(ctx context.Context, sku, email string) error {
	email, ok := httpx.NormEmail(email)
	if !ok {
		return httpx.InvalidEmail
	}
	tag, err := s.db.Exec(ctx, `insert into stock_notifications (size_id, email) select id, $2 from product_sizes where sku = $1 on conflict do nothing`, sku, email)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		_ = s.db.QueryRow(ctx, `select exists(select 1 from product_sizes where sku = $1)`, sku).Scan(&exists)
		if !exists {
			return httpx.ErrNotFound
		}
	}
	return nil
}

func (s *Store) ListStockAlerts(ctx context.Context) ([]StockAlert, error) {
	rows, err := s.db.Query(ctx, `select n.id, n.size_id, s.sku, p.name, n.email, n.sent_at, n.created_at
		from stock_notifications n join product_sizes s on s.id = n.size_id join products p on p.id = s.product_id order by n.created_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StockAlert{}
	for rows.Next() {
		var a StockAlert
		if err := rows.Scan(&a.ID, &a.SizeID, &a.SKU, &a.ProductName, &a.Email, &a.SentAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DueStockAlerts returns pending alerts whose size is back in stock and marks them sent.
func (s *Store) DueStockAlerts(ctx context.Context) ([]StockAlert, error) {
	rows, err := s.db.Query(ctx, `with due as (
		update stock_notifications n set sent_at = now() from product_sizes s where s.id = n.size_id and n.sent_at is null and s.stock > 0
		returning n.id, n.size_id, n.email, n.created_at, s.sku, s.product_id)
		select d.id, d.size_id, d.sku, p.name || '|' || p.slug, d.email, now(), d.created_at from due d join products p on p.id = d.product_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StockAlert{}
	for rows.Next() {
		var a StockAlert
		var sent time.Time
		if err := rows.Scan(&a.ID, &a.SizeID, &a.SKU, &a.ProductName, &a.Email, &sent, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.SentAt = &sent
		out = append(out, a)
	}
	return out, rows.Err()
}

// ── Messages ─────────────────────────────────────────────────────────────────

const msgCols = `id, user_id, order_id, name, email, subject, body, status, reply, replied_at, created_at`

func scanMsg(row interface{ Scan(...any) error }) (Message, error) {
	var m Message
	err := row.Scan(&m.ID, &m.UserID, &m.OrderID, &m.Name, &m.Email, &m.Subject, &m.Body, &m.Status, &m.Reply, &m.RepliedAt, &m.CreatedAt)
	return m, err
}

func (s *Store) CreateMessage(ctx context.Context, m Message) (Message, error) {
	m.Name, m.Body = strings.TrimSpace(m.Name), strings.TrimSpace(m.Body)
	email, ok := httpx.NormEmail(m.Email)
	if m.Name == "" || m.Body == "" {
		return m, httpx.Unprocessable("missing_fields", "Name, email and message are required.")
	}
	if !ok {
		return m, httpx.InvalidEmail
	}
	m.Email = email
	if err := httpx.CheckLengths(httpx.Limit{Name: "Name", Value: m.Name, Max: 100}, httpx.Limit{Name: "Subject", Value: deref(m.Subject), Max: 150},
		httpx.Limit{Name: "Order number", Value: deref(m.OrderID), Max: 20}, httpx.Limit{Name: "Message", Value: m.Body, Max: 5000}); err != nil {
		return m, err
	}
	row := s.db.QueryRow(ctx, `insert into messages (user_id, order_id, name, email, subject, body) values ($1,$2,$3,$4,$5,$6) returning `+msgCols,
		m.UserID, m.OrderID, m.Name, m.Email, m.Subject, m.Body)
	return scanMsg(row)
}

func (s *Store) ListMessages(ctx context.Context, status string) ([]Message, error) {
	q := `select ` + msgCols + ` from messages`
	var args []any
	if status != "" {
		q += ` where status = $1`
		args = append(args, status)
	}
	rows, err := s.db.Query(ctx, q+` order by created_at desc limit 500`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		m, err := scanMsg(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) ReplyMessage(ctx context.Context, id int64, reply string) (Message, error) {
	row := s.db.QueryRow(ctx, `update messages set reply = $2, replied_at = now(), status = 'answered' where id = $1 returning `+msgCols, id, strings.TrimSpace(reply))
	m, err := scanMsg(row)
	return m, notFound(err)
}

func (s *Store) SetMessageStatus(ctx context.Context, id int64, status string) error {
	tag, err := s.db.Exec(ctx, `update messages set status = $2 where id = $1`, id, status)
	if err == nil && tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return err
}

// ── Content & blog ───────────────────────────────────────────────────────────

func (s *Store) ContentBlock(ctx context.Context, key string) (ContentBlock, error) {
	var c ContentBlock
	err := s.db.QueryRow(ctx, `select key, type, title, body, status, publish_at, updated_at from content_blocks where key = $1`, key).
		Scan(&c.Key, &c.Type, &c.Title, &c.Body, &c.Status, &c.PublishAt, &c.UpdatedAt)
	return c, notFound(err)
}

func (s *Store) ListContent(ctx context.Context) ([]ContentBlock, error) {
	rows, err := s.db.Query(ctx, `select key, type, title, body, status, publish_at, updated_at from content_blocks order by key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ContentBlock{}
	for rows.Next() {
		var c ContentBlock
		if err := rows.Scan(&c.Key, &c.Type, &c.Title, &c.Body, &c.Status, &c.PublishAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) UpsertContent(ctx context.Context, c ContentBlock) error {
	if c.Key == "" {
		return httpx.BadRequest("A key is required.")
	}
	if c.Type == "" {
		c.Type = "block"
	}
	if c.Status == "" {
		c.Status = "published"
	}
	if len(c.Body) == 0 || !json.Valid(c.Body) {
		c.Body = []byte("{}")
	}
	_, err := s.db.Exec(ctx, `insert into content_blocks (key, type, title, body, status, publish_at) values ($1,$2,$3,$4,$5,$6)
		on conflict (key) do update set type=excluded.type, title=excluded.title, body=excluded.body, status=excluded.status, publish_at=excluded.publish_at`,
		c.Key, c.Type, c.Title, c.Body, c.Status, c.PublishAt)
	return err
}

func (s *Store) DeleteContent(ctx context.Context, key string) error {
	_, err := s.db.Exec(ctx, `delete from content_blocks where key = $1`, key)
	return err
}

func (s *Store) FAQs(ctx context.Context) ([]FAQ, error) {
	c, err := s.ContentBlock(ctx, "faq")
	if err != nil {
		return []FAQ{}, nil
	}
	var body struct {
		Items []FAQ `json:"items"`
	}
	_ = json.Unmarshal(c.Body, &body)
	if body.Items == nil {
		body.Items = []FAQ{}
	}
	return body.Items, nil
}

const blogCols = `id, slug, title, excerpt, body, category, author, image_url, published, published_at, created_at`

func scanBlog(row interface{ Scan(...any) error }) (BlogPost, error) {
	var b BlogPost
	err := row.Scan(&b.ID, &b.Slug, &b.Title, &b.Excerpt, &b.Body, &b.Category, &b.Author, &b.ImageURL, &b.Published, &b.PublishedAt, &b.CreatedAt)
	return b, err
}

func (s *Store) ListBlog(ctx context.Context, all bool) ([]BlogPost, error) {
	q := `select ` + blogCols + ` from blog_posts`
	if !all {
		q += ` where published and (published_at is null or published_at <= now())`
	}
	rows, err := s.db.Query(ctx, q+` order by coalesce(published_at, created_at) desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlogPost{}
	for rows.Next() {
		b, err := scanBlog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) BlogBySlug(ctx context.Context, slug string) (BlogPost, error) {
	b, err := scanBlog(s.db.QueryRow(ctx, `select `+blogCols+` from blog_posts where slug = $1 and published`, slug))
	return b, notFound(err)
}

func (s *Store) UpsertBlog(ctx context.Context, b BlogPost) (BlogPost, error) {
	if b.Title == "" {
		return b, httpx.BadRequest("A title is required.")
	}
	if b.Slug == "" {
		b.Slug = strings.Trim(strings.ToLower(strings.ReplaceAll(b.Title, " ", "-")), "-")
	}
	if b.Author == "" {
		b.Author = "Celestial Oud"
	}
	if b.Published && b.PublishedAt == nil {
		now := nowUTC()
		b.PublishedAt = &now
	}
	if b.ID == 0 {
		err := s.db.QueryRow(ctx, `insert into blog_posts (slug, title, excerpt, body, category, author, image_url, published, published_at)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9) returning id, created_at`, b.Slug, b.Title, b.Excerpt, b.Body, b.Category, b.Author, b.ImageURL, b.Published, b.PublishedAt).
			Scan(&b.ID, &b.CreatedAt)
		return b, err
	}
	tag, err := s.db.Exec(ctx, `update blog_posts set slug=$2, title=$3, excerpt=$4, body=$5, category=$6, author=$7, image_url=$8, published=$9, published_at=$10 where id=$1`,
		b.ID, b.Slug, b.Title, b.Excerpt, b.Body, b.Category, b.Author, b.ImageURL, b.Published, b.PublishedAt)
	if err == nil && tag.RowsAffected() == 0 {
		err = httpx.ErrNotFound
	}
	return b, err
}

func (s *Store) DeleteBlog(ctx context.Context, id int64) error {
	_, err := s.db.Exec(ctx, `delete from blog_posts where id = $1`, id)
	return err
}

// ── Email template gate (mail.Gate) ──────────────────────────────────────────

func (s *Store) Template(ctx context.Context, key string) (mail.Template, error) {
	var t mail.Template
	err := s.db.QueryRow(ctx, `select key, subject, enabled from email_templates where key = $1`, key).Scan(&t.Key, &t.Subject, &t.Enabled)
	if err == pgx.ErrNoRows {
		// Admin notifications are gated by admin_notifications instead.
		var enabled bool
		if err2 := s.db.QueryRow(ctx, `select enabled from admin_notifications where key = $1`, key).Scan(&enabled); err2 == nil {
			return mail.Template{Key: key, Subject: "[Celestial Oud] " + strings.ReplaceAll(key, "_", " "), Enabled: enabled}, nil
		}
		return mail.Template{Key: key, Subject: key, Enabled: true}, nil
	}
	return t, err
}
