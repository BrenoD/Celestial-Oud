package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
	"github.com/stripe/stripe-go/v81/webhook"

	"github.com/BrenoD/Celestial-Oud/backend/internal/config"
)

// Payments wraps Stripe. When no key is configured (local dev, design review)
// every method behaves as a successful no-op so the checkout still works.
type Payments struct {
	sc      *client.API
	enabled bool
	secret  string
}

func NewPayments(cfg config.Config) *Payments {
	p := &Payments{enabled: cfg.StripeEnabled(), secret: cfg.StripeWebhookSecret}
	if p.enabled {
		p.sc = &client.API{}
		p.sc.Init(cfg.StripeSecretKey, nil)
	}
	return p
}

func (p *Payments) Enabled() bool { return p.enabled }

type Intent struct {
	ID           string `json:"id"`
	ClientSecret string `json:"clientSecret"`
	Amount       int    `json:"amount"`
}

// CreateIntent creates a PaymentIntent for an order. The Idempotency-Key is
// forwarded to Stripe, so a retried checkout reuses the same intent.
func (p *Payments) CreateIntent(ctx context.Context, orderID, email string, amount int, idem string) (*Intent, error) {
	if !p.enabled {
		return nil, nil
	}
	params := &stripe.PaymentIntentParams{
		Amount:                  stripe.Int64(int64(amount)),
		Currency:                stripe.String("gbp"),
		ReceiptEmail:            stripe.String(email),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{Enabled: stripe.Bool(true)},
		Metadata:                map[string]string{"order_id": orderID},
	}
	params.Context = ctx
	if idem != "" {
		params.SetIdempotencyKey("order_" + idem)
	}
	pi, err := p.sc.PaymentIntents.New(params)
	if err != nil {
		return nil, err
	}
	return &Intent{ID: pi.ID, ClientSecret: pi.ClientSecret, Amount: int(pi.Amount)}, nil
}

func (p *Payments) IntentStatus(ctx context.Context, id string) (string, error) {
	if !p.enabled {
		return "succeeded", nil
	}
	params := &stripe.PaymentIntentParams{}
	params.Context = ctx
	pi, err := p.sc.PaymentIntents.Get(id, params)
	if err != nil {
		return "", err
	}
	return string(pi.Status), nil
}

func (p *Payments) Refund(ctx context.Context, intentID string, amount int) error {
	if !p.enabled {
		return nil
	}
	params := &stripe.RefundParams{PaymentIntent: stripe.String(intentID)}
	if amount > 0 {
		params.Amount = stripe.Int64(int64(amount))
	}
	params.Context = ctx
	_, err := p.sc.Refunds.New(params)
	return err
}

type PaymentRow struct {
	Ref       string `json:"ref"`
	StripeID  string `json:"stripeId"`
	OrderID   string `json:"orderId"`
	Method    string `json:"method"`
	Email     string `json:"email"`
	Amount    int    `json:"amount"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"createdAt"`
}

func (p *Payments) List(ctx context.Context, limit int) ([]PaymentRow, error) {
	if !p.enabled {
		return []PaymentRow{}, nil
	}
	params := &stripe.PaymentIntentListParams{}
	params.Context = ctx
	params.Limit = stripe.Int64(int64(limit))
	params.AddExpand("data.latest_charge")
	it := p.sc.PaymentIntents.List(params)
	out := []PaymentRow{}
	for it.Next() {
		pi := it.PaymentIntent()
		row := PaymentRow{Ref: pi.ID, StripeID: pi.ID, OrderID: pi.Metadata["order_id"], Email: pi.ReceiptEmail, Amount: int(pi.Amount), CreatedAt: pi.Created, Method: "card"}
		switch {
		case pi.LatestCharge != nil && pi.LatestCharge.Refunded:
			row.Status = "refunded"
		case pi.LatestCharge != nil && pi.LatestCharge.AmountRefunded > 0:
			row.Status = "partial_refund"
		case pi.Status == stripe.PaymentIntentStatusSucceeded:
			row.Status = "paid"
		case pi.Status == stripe.PaymentIntentStatusCanceled:
			row.Status = "canceled"
		default:
			row.Status = "pending"
		}
		if pi.LatestCharge != nil && pi.LatestCharge.PaymentMethodDetails != nil {
			row.Method = string(pi.LatestCharge.PaymentMethodDetails.Type)
		}
		out = append(out, row)
	}
	return out, it.Err()
}

// ParseWebhook verifies the signature and returns the event.
func (p *Payments) ParseWebhook(r *http.Request, body []byte) (stripe.Event, error) {
	if p.secret == "" {
		return stripe.Event{}, errors.New("webhook secret not configured")
	}
	return webhook.ConstructEventWithOptions(body, r.Header.Get("Stripe-Signature"), p.secret, webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
}
