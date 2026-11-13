// Package mail sends transactional email through Resend. Templates are keyed
// like rows in email_templates so the admin can switch them off.
package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Template struct {
	Key     string
	Subject string
	Enabled bool
}

// Gate decides whether a template may be sent (backed by email_templates).
type Gate interface {
	Template(ctx context.Context, key string) (Template, error)
}

type Mailer struct {
	apiKey   string
	from     string
	admins   []string
	siteURL  string
	gate     Gate
	log      *slog.Logger
	client   *http.Client
	disabled bool
}

func New(apiKey, from string, admins []string, siteURL string, gate Gate, log *slog.Logger) *Mailer {
	return &Mailer{
		apiKey: apiKey, from: from, admins: admins, siteURL: siteURL, gate: gate, log: log,
		client:   &http.Client{Timeout: 10 * time.Second},
		disabled: apiKey == "",
	}
}

// Send renders the keyed template and dispatches it in the background.
func (m *Mailer) Send(ctx context.Context, key, to string, data map[string]any) {
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := m.send(bg, key, to, data); err != nil {
			m.log.Warn("email not sent", "template", key, "to", to, "err", err)
		}
	}()
}

// Admin notifies the store team (ADMIN_EMAILS).
func (m *Mailer) Admin(ctx context.Context, key string, data map[string]any) {
	for _, a := range m.admins {
		m.Send(ctx, key, a, data)
	}
}

func (m *Mailer) send(ctx context.Context, key, to string, data map[string]any) error {
	tpl, err := m.gate.Template(ctx, key)
	if err != nil {
		return err
	}
	if !tpl.Enabled {
		return nil
	}
	subject := interpolate(tpl.Subject, data)
	body := render(key, data, m.siteURL)
	if m.disabled {
		m.log.Info("email (dry run)", "to", to, "subject", subject)
		return nil
	}
	return m.Raw(ctx, to, subject, body)
}

// Raw sends an already-rendered HTML email (bulk newsletter, replies).
func (m *Mailer) Raw(ctx context.Context, to, subject, htmlBody string) error {
	if m.disabled {
		m.log.Info("email (dry run)", "to", to, "subject", subject)
		return nil
	}
	payload, _ := json.Marshal(map[string]any{"from": m.from, "to": []string{to}, "subject": subject, "html": htmlBody})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(res.Body)
		return fmt.Errorf("resend %d: %s", res.StatusCode, buf.String())
	}
	return nil
}

func interpolate(s string, data map[string]any) string {
	for k, v := range data {
		s = strings.ReplaceAll(s, "{{"+k+"}}", fmt.Sprint(v))
	}
	return s
}

// ── Templates (brand: white background, espresso text, bronze accents) ───────

func Layout(title, body, siteURL string) string {
	return `<!doctype html><html><head><meta charset="utf-8"><title>` + html.EscapeString(title) + `</title></head>
<body style="margin:0;background:#F7F3EC;font-family:Jost,Arial,sans-serif;color:#2E1D15">
<table width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:32px 16px">
<table width="560" cellpadding="0" cellspacing="0" style="max-width:560px;width:100%">
<tr><td style="padding:0 0 20px;font-family:Georgia,serif;font-size:20px;letter-spacing:.08em">CELESTIAL OUD</td></tr>
<tr><td style="background:#FFFFFF;border:1px solid #ECE6DD;border-radius:6px;padding:32px 28px;font-size:15px;line-height:1.6">` + body + `</td></tr>
<tr><td style="padding:18px 0 0;text-align:center;font-size:12px;color:#5A4334">Celestial Oud · <a href="` + siteURL + `" style="color:#7A5530">` + siteURL + `</a></td></tr>
</table></td></tr></table></body></html>`
}

func h1(s string) string {
	return `<h1 style="margin:0 0 12px;font-family:Georgia,serif;font-size:26px">` + html.EscapeString(s) + `</h1>`
}
func p(s string) string { return `<p style="margin:12px 0;color:#5A4334">` + s + `</p>` }
func btn(label, url string) string {
	return `<a href="` + url + `" style="display:inline-block;margin-top:16px;padding:14px 24px;background:#2E1D15;color:#FFFFFF;border-radius:4px;font-weight:600;letter-spacing:.08em;text-transform:uppercase;font-size:13px;text-decoration:none">` + html.EscapeString(label) + `</a>`
}

func str(data map[string]any, k string) string { return html.EscapeString(fmt.Sprint(data[k])) }

// raw is for values the caller has already built from escaped parts (items_html).
func raw(data map[string]any, k string) string {
	v, _ := data[k].(SafeHTML)
	return string(v)
}

// SafeHTML marks a fragment as already escaped.
type SafeHTML string

// Paragraphs escapes plain text and turns blank-line-separated blocks into paragraphs.
func Paragraphs(text string) string {
	var b strings.Builder
	for _, para := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
		if para = strings.TrimSpace(para); para != "" {
			b.WriteString(p(strings.ReplaceAll(html.EscapeString(para), "\n", "<br>")))
		}
	}
	return b.String()
}

// UnsubscribeFooter is appended to every marketing email.
func UnsubscribeFooter(site, token string) string {
	u := site + "/newsletter/unsubscribe?token=" + url.QueryEscape(token)
	return `<p style="margin:24px 0 0;font-size:12px;color:#5A4334">You receive this because you subscribed to Celestial Oud news. <a href="` +
		html.EscapeString(u) + `" style="color:#7A5530">Unsubscribe</a></p>`
}

func render(key string, d map[string]any, site string) string {
	switch key {
	case "welcome":
		return Layout("Welcome", h1("Welcome, "+str(d, "name"))+p("Your Celestial Oud account is ready. Track orders, save your wishlist and build your scent profile.")+btn("Shop fragrances", site+"/collections/all"), site)
	case "order_confirmation":
		return Layout("Order confirmed", h1("Thank you — order "+str(d, "order_id")+" is confirmed")+p("We've received your payment of <b>"+str(d, "total")+"</b>. We'll email you again when it's dispatched.")+p(raw(d, "items_html"))+btn("View order", str(d, "order_url")), site)
	case "order_dispatched":
		return Layout("On its way", h1("Order "+str(d, "order_id")+" is on its way")+p("Carrier: <b>"+str(d, "carrier")+"</b> · Tracking: <b>"+str(d, "tracking")+"</b>")+btn("Track order", site+"/help#track"), site)
	case "order_delivered":
		return Layout("Delivered", h1("Order "+str(d, "order_id")+" has arrived")+p("We hope you love it. A review helps other customers find their scent.")+btn("Write a review", site+"/account/orders"), site)
	case "order_refunded":
		return Layout("Refund", h1("Your refund for "+str(d, "order_id"))+p("We've refunded <b>"+str(d, "total")+"</b> to your original payment method. It can take 5–10 working days to appear."), site)
	case "password_reset":
		return Layout("Reset password", h1("Reset your password")+p("This link is valid for 1 hour. If you didn't ask for it, you can ignore this email.")+btn("Choose a new password", str(d, "url")), site)
	case "back_in_stock":
		return Layout("Back in stock", h1(str(d, "product")+" is back")+p("The size you wanted is available again.")+btn("Shop now", site+"/products/"+str(d, "slug")), site)
	case "newsletter_confirm":
		return Layout("Confirm", h1("Confirm your subscription")+p("Tap the button to start receiving Celestial Oud news and offers. If you didn't ask for this, ignore this email and nothing will happen.")+btn("Confirm subscription", str(d, "url")), site)
	case "newsletter_welcome":
		return Layout("Welcome", h1("Your 10% welcome code")+p("Use <b>WELCOME10</b> at checkout on your first order.")+btn("Shop bestsellers", site+"/collections/bestsellers"), site)
	case "review_request":
		return Layout("How is it?", h1("How are you enjoying "+str(d, "product")+"?")+p("A short review helps other customers.")+btn("Write a review", site+"/account/orders"), site)
	case "chat_reply":
		return Layout("Reply", h1("We replied to your message")+Paragraphs(fmt.Sprint(d["reply"]))+btn("Continue the chat", site+"/help#chat"), site)
	case "new_chat_message":
		return Layout("Live chat", h1("New live chat message")+p("<b>"+str(d, "name")+"</b> ("+str(d, "email")+")")+Paragraphs(fmt.Sprint(d["body"]))+btn("Open the inbox", site+"/admin/chats"), site)
	case "message_reply":
		return Layout("Reply", h1("Re: "+str(d, "subject"))+p(strings.ReplaceAll(str(d, "reply"), "\n", "<br>")), site)
	// Admin notifications
	case "new_order":
		return Layout("New order", h1("New order "+str(d, "order_id"))+p(str(d, "customer")+" · "+str(d, "total"))+p(raw(d, "items_html")), site)
	case "low_stock":
		return Layout("Low stock", h1("Low stock: "+str(d, "sku"))+p("Only "+str(d, "stock")+" left."), site)
	case "new_review":
		return Layout("New review", h1("Review awaiting moderation")+p(str(d, "author")+" rated "+str(d, "product")+" "+str(d, "rating")+"/5"), site)
	case "new_message":
		return Layout("New message", h1("New customer message")+p("<b>"+str(d, "name")+"</b> ("+str(d, "email")+"): "+str(d, "body")), site)
	case "payment_failed":
		return Layout("Payment failed", h1("Payment failed")+p("PaymentIntent "+str(d, "intent")+": "+str(d, "reason")), site)
	}
	return Layout(key, p(fmt.Sprint(d)), site)
}
