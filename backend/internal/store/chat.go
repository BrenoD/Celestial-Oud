package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
)

type Chat struct {
	ID              int64         `json:"id"`
	UserID          *string       `json:"userId,omitempty"`
	CustomerName    string        `json:"customerName"`
	CustomerEmail   string        `json:"customerEmail,omitempty"`
	Status          string        `json:"status"`
	LastMessageAt   time.Time     `json:"lastMessageAt"`
	CreatedAt       time.Time     `json:"createdAt"`
	Unread          int           `json:"unread"`
	LastMessage     *string       `json:"lastMessage,omitempty"`
	Messages        []ChatMessage `json:"messages,omitempty"`
	guestTokenHash  *string
	staffLastReadAt *time.Time
}

type ChatMessage struct {
	ID        int64     `json:"id"`
	Sender    string    `json:"sender"` // customer | agent
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

const chatCols = `c.id, c.user_id, c.customer_name, c.customer_email, c.status, c.last_message_at, c.created_at, c.guest_token_hash, c.staff_last_read_at`

func scanChat(row pgx.Row) (*Chat, error) {
	var c Chat
	err := row.Scan(&c.ID, &c.UserID, &c.CustomerName, &c.CustomerEmail, &c.Status, &c.LastMessageAt, &c.CreatedAt, &c.guestTokenHash, &c.staffLastReadAt)
	if err != nil {
		return nil, notFound(err)
	}
	return &c, nil
}

// GuestTokenHash exposes the stored hash for constant-time comparison.
func (c *Chat) GuestTokenHash() string { return deref(c.guestTokenHash) }

func (s *Store) ChatByID(ctx context.Context, id int64) (*Chat, error) {
	return scanChat(s.db.QueryRow(ctx, `select `+chatCols+` from chats c where c.id = $1`, id))
}

// OpenChatForUser returns the signed-in customer's open conversation, if any.
func (s *Store) OpenChatForUser(ctx context.Context, userID string) (*Chat, error) {
	return scanChat(s.db.QueryRow(ctx, `select `+chatCols+` from chats c where c.user_id = $1 order by c.status = 'open' desc, c.last_message_at desc limit 1`, userID))
}

// StartChat creates a conversation with its first customer message.
func (s *Store) StartChat(ctx context.Context, userID *string, name, email, guestHash, body string) (*Chat, error) {
	var id int64
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `insert into chats (user_id, customer_name, customer_email, guest_token_hash, customer_last_read_at)
			values ($1,$2,$3,nullif($4,''), now()) returning id`, userID, name, email, guestHash).Scan(&id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `insert into chat_messages (chat_id, sender, body) values ($1,'customer',$2)`, id, body)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.ChatByID(ctx, id)
}

// AddChatMessage appends a message, reopens a closed chat and bumps its
// ordering. It reports whether the team had read everything before this
// message (used to send one notification per burst, not one per message).
func (s *Store) AddChatMessage(ctx context.Context, chatID int64, sender string, agentID *string, body string) (ChatMessage, bool, error) {
	var m ChatMessage
	wasRead := false
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `select not exists(select 1 from chat_messages m join chats c on c.id = m.chat_id
			where m.chat_id = $1 and m.sender = 'customer' and (c.staff_last_read_at is null or m.created_at > c.staff_last_read_at))`, chatID).Scan(&wasRead); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `insert into chat_messages (chat_id, sender, agent_id, body) values ($1,$2,$3,$4) returning id, sender, body, created_at`,
			chatID, sender, agentID, body).Scan(&m.ID, &m.Sender, &m.Body, &m.CreatedAt); err != nil {
			return err
		}
		read := `customer_last_read_at = now()`
		if sender == "agent" {
			read = `staff_last_read_at = now()`
		}
		_, err := tx.Exec(ctx, `update chats set status = 'open', last_message_at = now(), `+read+` where id = $1`, chatID)
		return err
	})
	return m, wasRead, err
}

// ChatMessages returns messages after a given id (0 = all), oldest first.
func (s *Store) ChatMessages(ctx context.Context, chatID, after int64) ([]ChatMessage, error) {
	rows, err := s.db.Query(ctx, `select id, sender, body, created_at from chat_messages where chat_id = $1 and id > $2 order by id limit 500`, chatID, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatMessage{}
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.Sender, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MarkChatRead records that the customer or the team has seen the conversation.
func (s *Store) MarkChatRead(ctx context.Context, chatID int64, staff bool) {
	col := "customer_last_read_at"
	if staff {
		col = "staff_last_read_at"
	}
	_, _ = s.db.Exec(ctx, `update chats set `+col+` = now() where id = $1`, chatID)
}

// CustomerUnread counts agent replies the customer has not seen.
func (s *Store) CustomerUnread(ctx context.Context, chatID int64) int {
	var n int
	_ = s.db.QueryRow(ctx, `select count(*) from chat_messages m join chats c on c.id = m.chat_id
		where m.chat_id = $1 and m.sender = 'agent' and (c.customer_last_read_at is null or m.created_at > c.customer_last_read_at)`, chatID).Scan(&n)
	return n
}

// CustomerActiveRecently says whether the customer read the chat in the last
// two minutes (then they will see the reply live: no email needed).
func (s *Store) CustomerActiveRecently(ctx context.Context, chatID int64) bool {
	var active bool
	_ = s.db.QueryRow(ctx, `select coalesce(customer_last_read_at > now() - interval '2 minutes', false) from chats where id = $1`, chatID).Scan(&active)
	return active
}

func (s *Store) SetChatStatus(ctx context.Context, chatID int64, status string) error {
	if status != "open" && status != "closed" {
		return httpx.BadRequest("Status must be open or closed.")
	}
	tag, err := s.db.Exec(ctx, `update chats set status = $2 where id = $1`, chatID, status)
	if err == nil && tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return err
}

// ListChats is the team inbox: unread (shared across staff) and the last message.
func (s *Store) ListChats(ctx context.Context, status string) ([]Chat, error) {
	rows, err := s.db.Query(ctx, `select `+chatCols+`,
		(select count(*) from chat_messages m where m.chat_id = c.id and m.sender = 'customer' and (c.staff_last_read_at is null or m.created_at > c.staff_last_read_at)),
		(select body from chat_messages m where m.chat_id = c.id order by m.id desc limit 1)
		from chats c where ($1 = '' or c.status = $1) order by c.last_message_at desc limit 200`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Chat{}
	for rows.Next() {
		var c Chat
		if err := rows.Scan(&c.ID, &c.UserID, &c.CustomerName, &c.CustomerEmail, &c.Status, &c.LastMessageAt, &c.CreatedAt, &c.guestTokenHash, &c.staffLastReadAt, &c.Unread, &c.LastMessage); err != nil {
			return nil, err
		}
		if c.LastMessage != nil && len([]rune(*c.LastMessage)) > 120 {
			cut := string([]rune(*c.LastMessage)[:120]) + "…"
			c.LastMessage = &cut
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UnreadChats counts conversations with customer messages the team has not read.
func (s *Store) UnreadChats(ctx context.Context) int {
	var n int
	_ = s.db.QueryRow(ctx, `select count(*) from chats c where c.status = 'open' and exists (select 1 from chat_messages m
		where m.chat_id = c.id and m.sender = 'customer' and (c.staff_last_read_at is null or m.created_at > c.staff_last_read_at))`).Scan(&n)
	return n
}
