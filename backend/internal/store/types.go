// Package store is the data-access layer. Types here are the API's JSON shapes
// and mirror frontend/src/types.ts (camelCase, integer pence).
package store

import (
	"encoding/json"
	"time"
)

type User struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	FirstName         string     `json:"firstName"`
	LastName          string     `json:"lastName"`
	Phone             *string    `json:"phone,omitempty"`
	Role              string     `json:"role"`
	Tier              string     `json:"tier"`
	Points            int        `json:"points"`
	MarketingEmail    bool       `json:"marketingEmail"`
	MarketingSMS      bool       `json:"marketingSms"`
	MarketingWhatsApp bool       `json:"marketingWhatsapp"`
	EmailVerifiedAt   *time.Time `json:"emailVerifiedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	PasswordHash      string     `json:"-"`
}

type Address struct {
	ID        int64   `json:"id,omitempty"`
	Label     *string `json:"label,omitempty"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Line1     string  `json:"line1"`
	Line2     *string `json:"line2,omitempty"`
	City      string  `json:"city"`
	Postcode  string  `json:"postcode"`
	Country   string  `json:"country"`
	Phone     *string `json:"phone,omitempty"`
	IsDefault bool    `json:"isDefault,omitempty"`
}

type Collection struct {
	ID          int64   `json:"id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	ImageURL    *string `json:"image,omitempty"`
	Sort        int     `json:"sort"`
	Visible     bool    `json:"visible"`
}

type ProductSize struct {
	ID             int64  `json:"id,omitempty"`
	SKU            string `json:"sku"`
	Label          string `json:"label"`
	ML             int    `json:"ml"`
	Price          int    `json:"price"`
	CompareAtPrice *int   `json:"compareAtPrice,omitempty"`
	Stock          int    `json:"stock"`
	InStock        bool   `json:"inStock"`
	IsSample       bool   `json:"isSample"`
	Sort           int    `json:"sort"`
}

type Pyramid struct {
	Top   []string `json:"top"`
	Heart []string `json:"heart"`
	Base  []string `json:"base"`
}

type Profile struct {
	Intensity int `json:"intensity"`
	Longevity int `json:"longevity"`
	Sweetness int `json:"sweetness"`
	Freshness int `json:"freshness"`
}

type Product struct {
	ID                int64         `json:"id"`
	Slug              string        `json:"slug"`
	Name              string        `json:"name"`
	Tagline           string        `json:"tagline"`
	NotesLine         string        `json:"notesLine"`
	Badge             *string       `json:"badge,omitempty"`
	Tags              []string      `json:"tags"`
	Family            string        `json:"family"`
	Concentration     string        `json:"concentration"`
	KeyNotes          []string      `json:"keyNotes"`
	Wear              []string      `json:"wear"`
	Rating            float64       `json:"rating"`
	ReviewCount       int           `json:"reviewCount"`
	Images            []string      `json:"images"`
	Sizes             []ProductSize `json:"sizes"`
	SamplePrice       *int          `json:"samplePrice,omitempty"`
	Pyramid           Pyramid       `json:"pyramid"`
	Profile           Profile       `json:"profile"`
	BestFor           string        `json:"bestFor"`
	Description       string        `json:"description"`
	HowToWear         string        `json:"howToWear"`
	Ingredients       string        `json:"ingredients"`
	Allergens         []string      `json:"allergens"`
	ResponsiblePerson *string       `json:"responsiblePerson,omitempty"`
	CPSRRef           *string       `json:"cpsrRef,omitempty"`
	SCPNRef           *string       `json:"scpnRef,omitempty"`
	Status            string        `json:"status"`
	Sort              int           `json:"sort"`
	Collection        CollectionRef `json:"collection"`
	CollectionSlugs   []string      `json:"collectionSlugs"`
	CreatedAt         time.Time     `json:"createdAt"`
	UpdatedAt         time.Time     `json:"updatedAt"`
}

type CollectionRef struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type ProductSummary struct {
	ID          int64    `json:"id"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	NotesLine   string   `json:"notesLine"`
	Badge       *string  `json:"badge,omitempty"`
	Tags        []string `json:"tags"`
	Rating      float64  `json:"rating"`
	ReviewCount int      `json:"reviewCount"`
	Images      []string `json:"images"`
	FromPrice   int      `json:"fromPrice"`
	Status      string   `json:"status,omitempty"`
	TotalStock  int      `json:"totalStock,omitempty"`
}

type ProductQuery struct {
	Collection string
	Q          string
	Family     []string
	Notes      []string
	Size       []int
	Price      []string
	Wear       []string
	Sort       string
	Page       int
	PageSize   int
	// Admin: include drafts / archived
	AllStatuses bool
}

type Page[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type Review struct {
	ID          int64     `json:"id"`
	ProductID   int64     `json:"productId"`
	ProductName string    `json:"productName,omitempty"`
	ProductSlug string    `json:"productSlug,omitempty"`
	Author      string    `json:"author"`
	Rating      int       `json:"rating"`
	Title       *string   `json:"title"`
	Body        *string   `json:"body"`
	Longevity   *string   `json:"longevity,omitempty"`
	Verified    bool      `json:"verified"`
	Approved    bool      `json:"approved"`
	CreatedAt   time.Time `json:"createdAt"`
}

type DeliveryMethod struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ETA         string  `json:"eta"`
	Price       int     `json:"price"`
	FreeOver    *int    `json:"freeOver,omitempty"`
	Carrier     *string `json:"carrier,omitempty"`
	ServiceCode *string `json:"serviceCode,omitempty"`
	Active      bool    `json:"active"`
	Sort        int     `json:"sort"`
}

type DiscountCode struct {
	Code        string     `json:"code"`
	Type        string     `json:"type"`
	Value       int        `json:"value"`
	MinSubtotal int        `json:"minSubtotal"`
	MaxUses     *int       `json:"maxUses"`
	Uses        int        `json:"uses"`
	StartsAt    *time.Time `json:"startsAt"`
	EndsAt      *time.Time `json:"endsAt"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type OrderLine struct {
	SKU          string  `json:"sku"`
	ProductID    *int64  `json:"productId"`
	SizeID       *int64  `json:"sizeId,omitempty"`
	Slug         string  `json:"slug,omitempty"`
	Name         string  `json:"name"`
	VariantLabel string  `json:"variantLabel"`
	Image        *string `json:"image"`
	UnitPrice    int     `json:"unitPrice"`
	Qty          int     `json:"qty"`
}

type Tracking struct {
	Carrier string  `json:"carrier"`
	Number  string  `json:"number"`
	URL     *string `json:"url,omitempty"`
}

type Order struct {
	ID                 string  `json:"id"`
	UserID             *string `json:"userId,omitempty"`
	Email              string  `json:"email"`
	CustomerName       string  `json:"customerName"`
	Status             string  `json:"status"`
	PaymentStatus      string  `json:"paymentStatus"`
	Subtotal           int     `json:"subtotal"`
	Discount           int     `json:"discount"`
	Delivery           int     `json:"delivery"`
	Total              int     `json:"total"`
	Currency           string  `json:"currency"`
	DiscountCode       *string `json:"discountCode,omitempty"`
	DeliveryMethodID   string  `json:"deliveryMethodId"`
	DeliveryMethodName string  `json:"deliveryMethodName,omitempty"`
	ShippingAddress    Address `json:"shippingAddress"`
	FreeSample         *string `json:"freeSample,omitempty"`
	// AccessToken is returned once, when the order is created: the guest's
	// key to view it later. Only its hash is stored.
	AccessToken         string      `json:"accessToken,omitempty"`
	GiftWrap            bool        `json:"giftWrap"`
	GiftMessage         *string     `json:"giftMessage,omitempty"`
	Notes               *string     `json:"notes,omitempty"`
	Lines               []OrderLine `json:"lines"`
	Tracking            *Tracking   `json:"tracking,omitempty"`
	StripePaymentIntent *string     `json:"stripePaymentIntent,omitempty"`
	EstimatedDelivery   string      `json:"estimatedDelivery"`
	CreatedAt           time.Time   `json:"createdAt"`
	PaidAt              *time.Time  `json:"paidAt,omitempty"`
	DispatchedAt        *time.Time  `json:"dispatchedAt,omitempty"`
	DeliveredAt         *time.Time  `json:"deliveredAt,omitempty"`
}

type CreateOrderInput struct {
	Email          string `json:"email"`
	MarketingOptIn bool   `json:"marketingOptIn"`
	Lines          []struct {
		SKU string `json:"sku"`
		Qty int    `json:"qty"`
	} `json:"lines"`
	FreeSampleSKU    *string `json:"freeSampleSku,omitempty"`
	GiftWrap         bool    `json:"giftWrap"`
	GiftMessage      *string `json:"giftMessage,omitempty"`
	DiscountCode     *string `json:"discountCode,omitempty"`
	DeliveryMethodID string  `json:"deliveryMethodId"`
	ShippingAddress  Address `json:"shippingAddress"`
	PaymentToken     string  `json:"paymentToken,omitempty"`
	Notes            *string `json:"notes,omitempty"`
}

type Message struct {
	ID        int64      `json:"id"`
	UserID    *string    `json:"userId,omitempty"`
	OrderID   *string    `json:"orderId,omitempty"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Subject   *string    `json:"subject"`
	Body      string     `json:"body"`
	Status    string     `json:"status"`
	Reply     *string    `json:"reply,omitempty"`
	RepliedAt *time.Time `json:"repliedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type FAQ struct {
	Q string `json:"q"`
	A string `json:"a"`
}

type ContentBlock struct {
	Key       string          `json:"key"`
	Type      string          `json:"type"`
	Title     *string         `json:"title"`
	Body      json.RawMessage `json:"body"`
	Status    string          `json:"status"`
	PublishAt *time.Time      `json:"publishAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type BlogPost struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Excerpt     *string    `json:"excerpt"`
	Body        *string    `json:"body"`
	Category    *string    `json:"category"`
	Author      string     `json:"author"`
	ImageURL    *string    `json:"imageUrl"`
	Published   bool       `json:"published"`
	PublishedAt *time.Time `json:"publishedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type ScentProfile struct {
	Answers      json.RawMessage `json:"answers"`
	Families     []string        `json:"families"`
	LikedNotes   []string        `json:"likedNotes"`
	AvoidedNotes []string        `json:"avoidedNotes"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type Settings struct {
	StoreName             string  `json:"storeName"`
	SupportEmail          string  `json:"supportEmail"`
	Currency              string  `json:"currency"`
	FreeDeliveryThreshold int     `json:"freeDeliveryThreshold"`
	DispatchCutoff        string  `json:"dispatchCutoff"`
	ReturnsDays           int     `json:"returnsDays"`
	MaintenanceMode       bool    `json:"maintenanceMode"`
	CompanyName           *string `json:"companyName"`
	CompanyNumber         *string `json:"companyNumber"`
	RegisteredAddress     *string `json:"registeredAddress"`
	VATNumber             *string `json:"vatNumber"`
}

type StockAlert struct {
	ID          int64      `json:"id"`
	SizeID      int64      `json:"sizeId"`
	SKU         string     `json:"sku"`
	ProductName string     `json:"productName"`
	Email       string     `json:"email"`
	SentAt      *time.Time `json:"sentAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type NewsletterSub struct {
	Email          string     `json:"email"`
	Name           *string    `json:"name"`
	Source         *string    `json:"source"`
	ConfirmedAt    *time.Time `json:"confirmedAt"`
	UnsubscribedAt *time.Time `json:"unsubscribedAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	OrderCount     int        `json:"orderCount"`
	Tier           string     `json:"tier"`
}

type Customer struct {
	User
	OrderCount int `json:"orderCount"`
	TotalSpent int `json:"totalSpent"`
}

type ScheduledPromotion struct {
	ID           int64      `json:"id"`
	ProductID    *int64     `json:"productId"`
	ProductName  *string    `json:"productName"`
	CollectionID *int64     `json:"collectionId"`
	PercentOff   int        `json:"percentOff"`
	StartsAt     time.Time  `json:"startsAt"`
	EndsAt       time.Time  `json:"endsAt"`
	AppliedAt    *time.Time `json:"appliedAt"`
	RevertedAt   *time.Time `json:"revertedAt"`
}
