package api

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/auth"
	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/store"
)

// Live chat. Customers (signed in, or guests holding a secret token sent in
// the X-Chat-Token header — never in the URL) talk to the team; staff with
// role support+ answer from the admin inbox. Clients poll; the API keeps each
// poll cheap (messages after the last id they have).

const chatTokenHeader = "X-Chat-Token"

func chatBody(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", httpx.Unprocessable("empty_message", "Write a message first.")
	}
	return s, httpx.CheckLengths(httpx.Limit{Name: "Message", Value: s, Max: 2000})
}

// chatAccess loads a chat the caller owns: the signed-in owner, or the guest
// whose token hashes to the stored value. Everyone else gets 404.
func (a *API) chatAccess(r *http.Request) (*store.Chat, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return nil, httpx.ErrNotFound
	}
	c, err := a.st.ChatByID(r.Context(), id)
	if err != nil {
		return nil, httpx.ErrNotFound
	}
	if u := httpx.UserFrom(r); u != nil && c.UserID != nil && *c.UserID == u.ID {
		return c, nil
	}
	if tok := r.Header.Get(chatTokenHeader); tok != "" && c.GuestTokenHash() != "" &&
		subtle.ConstantTimeCompare([]byte(auth.HashToken(tok)), []byte(c.GuestTokenHash())) == 1 {
		return c, nil
	}
	return nil, httpx.ErrNotFound
}

type chatView struct {
	ID       int64               `json:"id"`
	Status   string              `json:"status"`
	Name     string              `json:"customerName"`
	Messages []store.ChatMessage `json:"messages"`
	Unread   int                 `json:"unread"`
	Token    string              `json:"token,omitempty"` // returned once, to guests, when the chat starts
}

// startChat opens a conversation with its first message. Signed-in customers
// reuse their existing conversation instead of starting another.
func (a *API) startChat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name, Email, Message string
	}
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	body, err := chatBody(in.Message)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	u := httpx.UserFrom(r)
	if u != nil {
		if c, err := a.st.OpenChatForUser(r.Context(), u.ID); err == nil {
			a.postCustomerMessage(w, r, c, body)
			return
		}
		me, err := a.st.UserByID(r.Context(), u.ID)
		if err != nil {
			httpx.Fail(w, r, httpx.ErrUnauthorized)
			return
		}
		in.Email, in.Name = me.Email, strings.TrimSpace(me.FirstName+" "+me.LastName)
	}
	email, ok := httpx.NormEmail(in.Email)
	if !ok {
		httpx.Fail(w, r, httpx.InvalidEmail)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		httpx.Fail(w, r, httpx.Unprocessable("missing_name", "Enter your name."))
		return
	}
	if err := httpx.CheckLengths(httpx.Limit{Name: "Name", Value: in.Name, Max: 60}); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	plain, hash := "", ""
	if u == nil {
		plain, hash = auth.NewOpaqueToken()
	}
	c, err := a.st.StartChat(r.Context(), userID(r), in.Name, email, hash, body)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.mail.Admin(r.Context(), "new_chat_message", map[string]any{"name": c.CustomerName, "email": c.CustomerEmail, "body": body, "chat_id": c.ID})
	msgs, _ := a.st.ChatMessages(r.Context(), c.ID, 0)
	httpx.JSON(w, 201, chatView{ID: c.ID, Status: c.Status, Name: c.CustomerName, Messages: msgs, Token: plain})
}

// myChat returns the signed-in customer's conversation (404 when none).
func (a *API) myChat(w http.ResponseWriter, r *http.Request) {
	c, err := a.st.OpenChatForUser(r.Context(), httpx.UserFrom(r).ID)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrNotFound)
		return
	}
	msgs, _ := a.st.ChatMessages(r.Context(), c.ID, 0)
	httpx.JSON(w, 200, chatView{ID: c.ID, Status: c.Status, Name: c.CustomerName, Messages: msgs, Unread: a.st.CustomerUnread(r.Context(), c.ID)})
}

// getChat: messages after ?after=<id>; ?read=1 marks them seen (window open).
func (a *API) getChat(w http.ResponseWriter, r *http.Request) {
	c, err := a.chatAccess(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	msgs, err := a.st.ChatMessages(r.Context(), c.ID, after)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if r.URL.Query().Get("read") == "1" {
		a.st.MarkChatRead(r.Context(), c.ID, false)
	}
	httpx.JSON(w, 200, chatView{ID: c.ID, Status: c.Status, Name: c.CustomerName, Messages: msgs, Unread: a.st.CustomerUnread(r.Context(), c.ID)})
}

func (a *API) sendChatMessage(w http.ResponseWriter, r *http.Request) {
	c, err := a.chatAccess(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	var in struct{ Message string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	body, err := chatBody(in.Message)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.postCustomerMessage(w, r, c, body)
}

func (a *API) postCustomerMessage(w http.ResponseWriter, r *http.Request, c *store.Chat, body string) {
	// Per conversation: a person types, a script floods.
	if !a.allow(w, r, "chat-msg", strconv.FormatInt(c.ID, 10)) {
		return
	}
	m, teamWasUpToDate, err := a.st.AddChatMessage(r.Context(), c.ID, "customer", nil, body)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	// One email per burst: only when the team had read everything before.
	if teamWasUpToDate {
		a.mail.Admin(r.Context(), "new_chat_message", map[string]any{"name": c.CustomerName, "email": c.CustomerEmail, "body": body, "chat_id": c.ID})
	}
	httpx.JSON(w, 201, m)
}

// ── Admin inbox (support+) ───────────────────────────────────────────────────

func (a *API) adminChats(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && status != "open" && status != "closed" {
		status = ""
	}
	cs, err := a.st.ListChats(r.Context(), status)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, cs)
}

func (a *API) adminChat(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	c, err := a.st.ChatByID(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	c.Messages, err = a.st.ChatMessages(r.Context(), id, after)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	// Opening a conversation marks it read for the whole team.
	a.st.MarkChatRead(r.Context(), id, true)
	httpx.JSON(w, 200, c)
}

func (a *API) adminChatReply(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	c, err := a.st.ChatByID(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	var in struct{ Message string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	body, err := chatBody(in.Message)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	m, _, err := a.st.AddChatMessage(r.Context(), id, "agent", userID(r), body)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	// Email only if the customer is not watching the chat right now.
	if !a.st.CustomerActiveRecently(r.Context(), id) {
		a.mail.Send(r.Context(), "chat_reply", c.CustomerEmail, map[string]any{"name": c.CustomerName, "reply": body})
	}
	a.audit(r, "reply", "chat", strconv.FormatInt(id, 10), nil)
	httpx.JSON(w, 201, m)
}

func (a *API) adminChatStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.SetChatStatus(r.Context(), id, in.Status); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}
