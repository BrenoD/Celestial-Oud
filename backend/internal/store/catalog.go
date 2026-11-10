package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
)

// ── Collections ──────────────────────────────────────────────────────────────

const collectionCols = `id, slug, name, coalesce(description,''), image_url, sort, visible`

func scanCollection(row interface{ Scan(...any) error }) (Collection, error) {
	var c Collection
	err := row.Scan(&c.ID, &c.Slug, &c.Name, &c.Description, &c.ImageURL, &c.Sort, &c.Visible)
	return c, err
}

func (s *Store) ListCollections(ctx context.Context, includeHidden bool) ([]Collection, error) {
	q := `select ` + collectionCols + ` from collections`
	if !includeHidden {
		q += ` where visible`
	}
	rows, err := s.db.Query(ctx, q+` order by sort, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Collection{}
	for rows.Next() {
		c, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Virtual collections are computed from tags, not stored rows.
var virtualCollections = map[string]Collection{
	"all":         {Slug: "all", Name: "All Fragrances", Description: "Long-lasting oud, amber and musk perfumes.", Visible: true},
	"bestsellers": {Slug: "bestsellers", Name: "Bestsellers", Description: "Our most-loved fragrances.", Visible: true},
	"new-in":      {Slug: "new-in", Name: "New In", Description: "The latest additions to the collection.", Visible: true},
	"top-rated":   {Slug: "top-rated", Name: "Top rated", Description: "Our highest-rated fragrances, chosen by customers.", Visible: true},
	"offers":      {Slug: "offers", Name: "Offers", Description: "Limited-time prices.", Visible: true},
}

func (s *Store) CollectionBySlug(ctx context.Context, slug string) (Collection, error) {
	if v, ok := virtualCollections[slug]; ok {
		return v, nil
	}
	c, err := scanCollection(s.db.QueryRow(ctx, `select `+collectionCols+` from collections where slug = $1`, slug))
	return c, notFound(err)
}

func (s *Store) UpsertCollection(ctx context.Context, c Collection) (Collection, error) {
	if c.ID == 0 {
		err := s.db.QueryRow(ctx, `insert into collections (slug, name, description, image_url, sort, visible) values ($1,$2,$3,$4,$5,$6) returning id`,
			c.Slug, c.Name, c.Description, c.ImageURL, c.Sort, c.Visible).Scan(&c.ID)
		return c, err
	}
	tag, err := s.db.Exec(ctx, `update collections set slug=$2, name=$3, description=$4, image_url=$5, sort=$6, visible=$7 where id=$1`,
		c.ID, c.Slug, c.Name, c.Description, c.ImageURL, c.Sort, c.Visible)
	if err == nil && tag.RowsAffected() == 0 {
		err = httpx.ErrNotFound
	}
	return c, err
}

func (s *Store) DeleteCollection(ctx context.Context, id int64) error {
	_, err := s.db.Exec(ctx, `delete from collections where id = $1`, id)
	return err
}

// ── Products ─────────────────────────────────────────────────────────────────

// productSelect is a single query that aggregates sizes, notes, rating and the
// primary collection so list/detail never hit N+1.
const productSelect = `
select p.id, p.slug, p.name, p.tagline, p.notes_line, p.badge, p.tags, p.family, p.concentration, p.wear,
       p.images, p.pyramid, p.profile, p.best_for, p.description, p.how_to_wear, p.ingredients_inci, p.allergens,
       p.responsible_person, p.cpsr_ref, p.scpn_ref, p.status, p.sort, p.created_at, p.updated_at,
       coalesce((select json_agg(json_build_object('id',s.id,'sku',s.sku,'label',s.label,'ml',s.ml,'price',s.price,
                 'compareAtPrice',s.compare_at_price,'stock',s.stock,'isSample',s.is_sample,'sort',s.sort) order by s.sort, s.ml)
                 from product_sizes s where s.product_id = p.id), '[]'::json) as sizes,
       coalesce((select array_agg(n.slug order by n.name) from product_notes pn join notes n on n.id = pn.note_id
                 where pn.product_id = p.id and pn.tier = 'key'), '{}') as key_notes,
       coalesce((select round(avg(r.rating)::numeric, 1) from product_reviews r where r.product_id = p.id and r.approved), 0) as rating,
       (select count(*) from product_reviews r where r.product_id = p.id and r.approved) as review_count,
       coalesce((select array_agg(c.slug order by pc.sort, c.sort) from product_collections pc join collections c on c.id = pc.collection_id
                 where pc.product_id = p.id), '{}') as collection_slugs,
       coalesce((select c.name from product_collections pc join collections c on c.id = pc.collection_id
                 where pc.product_id = p.id order by pc.sort, c.sort limit 1), '') as collection_name
from products p`

type rawSize struct {
	ID             int64  `json:"id"`
	SKU            string `json:"sku"`
	Label          string `json:"label"`
	ML             int    `json:"ml"`
	Price          int    `json:"price"`
	CompareAtPrice *int   `json:"compareAtPrice"`
	Stock          int    `json:"stock"`
	IsSample       bool   `json:"isSample"`
	Sort           int    `json:"sort"`
}

func scanProduct(rows pgx.Rows) (*Product, error) {
	var p Product
	var sizesJSON, pyramidJSON, profileJSON, imagesJSON []byte
	var collectionName string
	err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Tagline, &p.NotesLine, &p.Badge, &p.Tags, &p.Family, &p.Concentration, &p.Wear,
		&imagesJSON, &pyramidJSON, &profileJSON, &p.BestFor, &p.Description, &p.HowToWear, &p.Ingredients, &p.Allergens,
		&p.ResponsiblePerson, &p.CPSRRef, &p.SCPNRef, &p.Status, &p.Sort, &p.CreatedAt, &p.UpdatedAt,
		&sizesJSON, &p.KeyNotes, &p.Rating, &p.ReviewCount, &p.CollectionSlugs, &collectionName)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(imagesJSON, &p.Images)
	_ = json.Unmarshal(pyramidJSON, &p.Pyramid)
	_ = json.Unmarshal(profileJSON, &p.Profile)
	var raw []rawSize
	_ = json.Unmarshal(sizesJSON, &raw)
	p.Sizes = make([]ProductSize, 0, len(raw))
	for _, r := range raw {
		sz := ProductSize{ID: r.ID, SKU: r.SKU, Label: r.Label, ML: r.ML, Price: r.Price, CompareAtPrice: r.CompareAtPrice,
			Stock: r.Stock, InStock: r.Stock > 0, IsSample: r.IsSample, Sort: r.Sort}
		if r.IsSample {
			price := r.Price
			p.SamplePrice = &price
			continue // samples are offered via the upsell, not as a size option
		}
		p.Sizes = append(p.Sizes, sz)
	}
	if p.Images == nil {
		p.Images = []string{}
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if len(p.CollectionSlugs) > 0 {
		p.Collection = CollectionRef{Slug: p.CollectionSlugs[0], Name: collectionName}
	} else {
		p.Collection = CollectionRef{Slug: "all", Name: "All Fragrances"}
	}
	return &p, nil
}

func summarize(p *Product) ProductSummary {
	from, total := 0, 0
	for _, sz := range p.Sizes {
		if from == 0 || sz.Price < from {
			from = sz.Price
		}
		total += sz.Stock
	}
	return ProductSummary{ID: p.ID, Slug: p.Slug, Name: p.Name, NotesLine: p.NotesLine, Badge: p.Badge, Tags: p.Tags,
		Rating: p.Rating, ReviewCount: p.ReviewCount, Images: p.Images, FromPrice: from, Status: p.Status, TotalStock: total}
}

func (s *Store) ProductBySlug(ctx context.Context, slug string, allStatuses bool) (*Product, error) {
	q := productSelect + ` where p.slug = $1`
	if !allStatuses {
		q += ` and p.status = 'active'`
	}
	rows, err := s.db.Query(ctx, q, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, httpx.ErrNotFound
	}
	return scanProduct(rows)
}

func (s *Store) ProductByID(ctx context.Context, id int64) (*Product, error) {
	rows, err := s.db.Query(ctx, productSelect+` where p.id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, httpx.ErrNotFound
	}
	return scanProduct(rows)
}

// ListProducts implements the storefront filter contract.
func (s *Store) ListProducts(ctx context.Context, q ProductQuery) (Page[ProductSummary], error) {
	var where []string
	var args []any
	add := func(cond string, v ...any) {
		for range v {
			args = append(args, v...)
			break
		}
		// renumber placeholders
		for i := range v {
			cond = strings.Replace(cond, fmt.Sprintf("$%d", i+1), fmt.Sprintf("$%d", len(args)-len(v)+i+1), 1)
		}
		where = append(where, cond)
	}
	if !q.AllStatuses {
		where = append(where, `p.status = 'active'`)
	}
	switch q.Collection {
	case "", "all":
	case "bestsellers":
		where = append(where, `'bestseller' = any(p.tags)`)
	case "new-in":
		where = append(where, `('new' = any(p.tags) or p.created_at > now() - interval '60 days')`)
	case "top-rated":
		where = append(where, `('top-rated' = any(p.tags) or (select count(*) from product_reviews r where r.product_id = p.id and r.approved) >= 5)`)
	case "offers":
		where = append(where, `('offer' = any(p.tags) or exists(select 1 from product_sizes s where s.product_id = p.id and s.compare_at_price > s.price))`)
	case "oud":
		where = append(where, `('oud' = any(p.tags) or exists(select 1 from product_collections pc join collections c on c.id = pc.collection_id where pc.product_id = p.id and c.slug = 'oud'))`)
	default:
		add(`exists(select 1 from product_collections pc join collections c on c.id = pc.collection_id where pc.product_id = p.id and c.slug = $1)`, q.Collection)
	}
	if q.Q != "" {
		add(`(p.name ilike '%' || $1 || '%' or p.notes_line ilike '%' || $1 || '%' or p.tagline ilike '%' || $1 || '%'
			or exists(select 1 from product_notes pn join notes n on n.id = pn.note_id where pn.product_id = p.id and n.slug ilike '%' || $1 || '%'))`, q.Q)
	}
	if len(q.Family) > 0 {
		add(`p.family = any($1)`, q.Family)
	}
	if len(q.Notes) > 0 {
		add(`exists(select 1 from product_notes pn join notes n on n.id = pn.note_id where pn.product_id = p.id and n.slug = any($1))`, q.Notes)
	}
	if len(q.Size) > 0 {
		add(`exists(select 1 from product_sizes s where s.product_id = p.id and s.ml = any($1))`, q.Size)
	}
	if len(q.Wear) > 0 {
		add(`p.wear && $1`, q.Wear)
	}
	if len(q.Price) > 0 {
		var bands []string
		for _, b := range q.Price {
			lo, hi, ok := strings.Cut(b, "-")
			if !ok {
				continue
			}
			cond := `exists(select 1 from product_sizes s where s.product_id = p.id and not s.is_sample`
			if lo != "" {
				cond += fmt.Sprintf(` and s.price >= %d`, atoi(lo))
			}
			if hi != "" {
				cond += fmt.Sprintf(` and s.price <= %d`, atoi(hi))
			}
			bands = append(bands, cond+`)`)
		}
		if len(bands) > 0 {
			where = append(where, "("+strings.Join(bands, " or ")+")")
		}
	}

	order := `(select count(*) from order_items oi where oi.product_id = p.id) desc, p.sort, p.id`
	switch q.Sort {
	case "newest":
		order = `p.created_at desc`
	case "price-asc":
		order = `(select min(price) from product_sizes s where s.product_id = p.id and not s.is_sample) asc nulls last`
	case "price-desc":
		order = `(select min(price) from product_sizes s where s.product_id = p.id and not s.is_sample) desc nulls last`
	case "top-rated":
		order = `rating desc, review_count desc`
	case "name":
		order = `p.name`
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " where " + strings.Join(where, " and ")
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 24
	}

	var total int
	if err := s.db.QueryRow(ctx, `select count(*) from products p`+whereSQL, args...).Scan(&total); err != nil {
		return Page[ProductSummary]{}, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := s.db.Query(ctx, productSelect+whereSQL+` order by `+order+fmt.Sprintf(` limit $%d offset $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return Page[ProductSummary]{}, err
	}
	defer rows.Close()
	items := []ProductSummary{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return Page[ProductSummary]{}, err
		}
		items = append(items, summarize(p))
	}
	return Page[ProductSummary]{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, rows.Err()
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func (s *Store) RelatedProducts(ctx context.Context, slug string, limit int) ([]ProductSummary, error) {
	rows, err := s.db.Query(ctx, productSelect+` where p.status = 'active' and p.slug <> $1
		order by (exists(select 1 from product_collections a join product_collections b on a.collection_id = b.collection_id
		          join products q on q.id = a.product_id where q.slug = $1 and b.product_id = p.id)) desc, p.sort limit $2`, slug, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProductSummary{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, summarize(p))
	}
	return out, rows.Err()
}

// ProductInput is the admin create/update payload.
type ProductInput struct {
	Slug              string        `json:"slug"`
	Name              string        `json:"name"`
	Tagline           string        `json:"tagline"`
	NotesLine         string        `json:"notesLine"`
	Badge             *string       `json:"badge"`
	Tags              []string      `json:"tags"`
	Family            string        `json:"family"`
	Concentration     string        `json:"concentration"`
	Wear              []string      `json:"wear"`
	Images            []string      `json:"images"`
	Pyramid           Pyramid       `json:"pyramid"`
	Profile           Profile       `json:"profile"`
	BestFor           string        `json:"bestFor"`
	Description       string        `json:"description"`
	HowToWear         string        `json:"howToWear"`
	Ingredients       string        `json:"ingredients"`
	Allergens         []string      `json:"allergens"`
	ResponsiblePerson *string       `json:"responsiblePerson"`
	CPSRRef           *string       `json:"cpsrRef"`
	SCPNRef           *string       `json:"scpnRef"`
	Status            string        `json:"status"`
	Sort              int           `json:"sort"`
	KeyNotes          []string      `json:"keyNotes"`
	CollectionSlugs   []string      `json:"collectionSlugs"`
	Sizes             []ProductSize `json:"sizes"`
}

func (in *ProductInput) normalise() error {
	in.Slug = strings.Trim(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(in.Slug), " ", "-")), "-")
	if in.Slug == "" {
		in.Slug = strings.Trim(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(in.Name), " ", "-")), "-")
	}
	if in.Name == "" {
		return httpx.BadRequest("A name is required.")
	}
	if in.Family == "" {
		in.Family = "woody-oud"
	}
	if in.Concentration == "" {
		in.Concentration = "Eau de Parfum"
	}
	if in.Status == "" {
		in.Status = "draft"
	}
	for _, s := range []*[]string{&in.Tags, &in.Wear, &in.Images, &in.Allergens, &in.KeyNotes, &in.CollectionSlugs} {
		if *s == nil {
			*s = []string{}
		}
	}
	for _, s := range []*[]string{&in.Pyramid.Top, &in.Pyramid.Heart, &in.Pyramid.Base} {
		if *s == nil {
			*s = []string{}
		}
	}
	if in.Profile == (Profile{}) {
		in.Profile = Profile{50, 50, 50, 50}
	}
	for i := range in.Sizes {
		in.Sizes[i].SKU = strings.TrimSpace(in.Sizes[i].SKU)
		if in.Sizes[i].SKU == "" {
			in.Sizes[i].SKU = fmt.Sprintf("%s-%s", in.Slug, strings.ToLower(strings.ReplaceAll(in.Sizes[i].Label, " ", "-")))
		}
		if in.Sizes[i].Price < 0 || in.Sizes[i].Stock < 0 {
			return httpx.BadRequest("Price and stock must not be negative.")
		}
	}
	return nil
}

// SaveProduct creates (id == 0) or updates a product with its sizes, notes and
// collections in one transaction. Sizes absent from the payload are deleted
// unless referenced by orders (then stock is set to 0).
func (s *Store) SaveProduct(ctx context.Context, id int64, in ProductInput) (*Product, error) {
	if err := in.normalise(); err != nil {
		return nil, err
	}
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		args := []any{in.Slug, in.Name, in.Tagline, in.NotesLine, in.Badge, in.Tags, in.Family, in.Concentration, in.Wear,
			jsonb(in.Images), jsonb(in.Pyramid), jsonb(in.Profile), in.BestFor, in.Description, in.HowToWear, in.Ingredients, in.Allergens,
			in.ResponsiblePerson, in.CPSRRef, in.SCPNRef, in.Status, in.Sort}
		if id == 0 {
			if err := tx.QueryRow(ctx, `insert into products (slug,name,tagline,notes_line,badge,tags,family,concentration,wear,images,pyramid,profile,
				best_for,description,how_to_wear,ingredients_inci,allergens,responsible_person,cpsr_ref,scpn_ref,status,sort)
				values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22) returning id`, args...).Scan(&id); err != nil {
				return err
			}
		} else {
			tag, err := tx.Exec(ctx, `update products set slug=$1,name=$2,tagline=$3,notes_line=$4,badge=$5,tags=$6,family=$7,concentration=$8,wear=$9,
				images=$10,pyramid=$11,profile=$12,best_for=$13,description=$14,how_to_wear=$15,ingredients_inci=$16,allergens=$17,
				responsible_person=$18,cpsr_ref=$19,scpn_ref=$20,status=$21,sort=$22 where id=$23`, append(args, id)...)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return httpx.ErrNotFound
			}
		}
		// Sizes
		keep := make([]string, 0, len(in.Sizes))
		for _, sz := range in.Sizes {
			keep = append(keep, sz.SKU)
			if _, err := tx.Exec(ctx, `insert into product_sizes (product_id, sku, label, ml, price, compare_at_price, stock, is_sample, sort)
				values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
				on conflict (sku) do update set product_id = excluded.product_id, label = excluded.label, ml = excluded.ml, price = excluded.price,
				compare_at_price = excluded.compare_at_price, stock = excluded.stock, is_sample = excluded.is_sample, sort = excluded.sort`,
				id, sz.SKU, sz.Label, sz.ML, sz.Price, sz.CompareAtPrice, sz.Stock, sz.IsSample, sz.Sort); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `delete from product_sizes where product_id = $1 and not (sku = any($2))
			and not exists (select 1 from order_items oi where oi.size_id = product_sizes.id)`, id, keep); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `update product_sizes set stock = 0 where product_id = $1 and not (sku = any($2))`, id, keep); err != nil {
			return err
		}
		// Key notes (create unknown notes on the fly)
		if _, err := tx.Exec(ctx, `delete from product_notes where product_id = $1 and tier = 'key'`, id); err != nil {
			return err
		}
		for _, n := range in.KeyNotes {
			slug := strings.ToLower(strings.TrimSpace(n))
			if slug == "" {
				continue
			}
			if _, err := tx.Exec(ctx, `insert into notes (slug, name) values ($1, initcap($1)) on conflict (slug) do nothing`, slug); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `insert into product_notes (product_id, note_id, tier) select $1, id, 'key' from notes where slug = $2 on conflict do nothing`, id, slug); err != nil {
				return err
			}
		}
		// Collections
		if _, err := tx.Exec(ctx, `delete from product_collections where product_id = $1`, id); err != nil {
			return err
		}
		for i, slug := range in.CollectionSlugs {
			if _, err := tx.Exec(ctx, `insert into product_collections (product_id, collection_id, sort) select $1, id, $3 from collections where slug = $2 on conflict do nothing`, id, slug, i); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "products_slug_key") {
			return nil, httpx.E(409, "slug_taken", "A product with this slug already exists.")
		}
		return nil, err
	}
	return s.ProductByID(ctx, id)
}

func (s *Store) DeleteProduct(ctx context.Context, id int64) error {
	// Soft delete when the product has ever been ordered; hard delete otherwise.
	var ordered bool
	if err := s.db.QueryRow(ctx, `select exists(select 1 from order_items where product_id = $1)`, id).Scan(&ordered); err != nil {
		return err
	}
	if ordered {
		_, err := s.db.Exec(ctx, `update products set status = 'archived' where id = $1`, id)
		return err
	}
	tag, err := s.db.Exec(ctx, `delete from products where id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

func (s *Store) AdjustStock(ctx context.Context, sku string, delta int) (int, error) {
	var stock int
	err := s.db.QueryRow(ctx, `update product_sizes set stock = greatest(0, stock + $2) where sku = $1 returning stock`, sku, delta).Scan(&stock)
	return stock, notFound(err)
}

func (s *Store) ListNotes(ctx context.Context) ([]map[string]string, error) {
	rows, err := s.db.Query(ctx, `select slug, name from notes order by name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var slug, name string
		if err := rows.Scan(&slug, &name); err != nil {
			return nil, err
		}
		out = append(out, map[string]string{"slug": slug, "name": name})
	}
	return out, rows.Err()
}

// ── Delivery methods & discounts ─────────────────────────────────────────────

func (s *Store) ListDeliveryMethods(ctx context.Context, all bool) ([]DeliveryMethod, error) {
	q := `select id, name, eta, price, free_over, carrier, service_code, active, sort from delivery_methods`
	if !all {
		q += ` where active`
	}
	rows, err := s.db.Query(ctx, q+` order by sort, price`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeliveryMethod{}
	for rows.Next() {
		var m DeliveryMethod
		if err := rows.Scan(&m.ID, &m.Name, &m.ETA, &m.Price, &m.FreeOver, &m.Carrier, &m.ServiceCode, &m.Active, &m.Sort); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) DeliveryMethod(ctx context.Context, id string) (DeliveryMethod, error) {
	var m DeliveryMethod
	err := s.db.QueryRow(ctx, `select id, name, eta, price, free_over, carrier, service_code, active, sort from delivery_methods where id = $1 and active`, id).
		Scan(&m.ID, &m.Name, &m.ETA, &m.Price, &m.FreeOver, &m.Carrier, &m.ServiceCode, &m.Active, &m.Sort)
	if err != nil {
		return m, httpx.Unprocessable("invalid_delivery", "Choose a valid delivery method.")
	}
	return m, nil
}

func (s *Store) UpsertDeliveryMethod(ctx context.Context, m DeliveryMethod) error {
	if m.ID == "" {
		m.ID = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(m.Name), " ", "-"))
	}
	_, err := s.db.Exec(ctx, `insert into delivery_methods (id, name, eta, price, free_over, carrier, service_code, active, sort)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9) on conflict (id) do update set name=excluded.name, eta=excluded.eta, price=excluded.price,
		free_over=excluded.free_over, carrier=excluded.carrier, service_code=excluded.service_code, active=excluded.active, sort=excluded.sort`,
		m.ID, m.Name, m.ETA, m.Price, m.FreeOver, m.Carrier, m.ServiceCode, m.Active, m.Sort)
	return err
}

func (s *Store) DeleteDeliveryMethod(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `delete from delivery_methods where id = $1`, id)
	return err
}

const discountCols = `code, type, value, min_subtotal, max_uses, uses, starts_at, ends_at, active, created_at`

func scanDiscount(row interface{ Scan(...any) error }) (DiscountCode, error) {
	var d DiscountCode
	err := row.Scan(&d.Code, &d.Type, &d.Value, &d.MinSubtotal, &d.MaxUses, &d.Uses, &d.StartsAt, &d.EndsAt, &d.Active, &d.CreatedAt)
	return d, err
}

// ValidateDiscount returns the discount amount for a subtotal, or an error.
func (s *Store) ValidateDiscount(ctx context.Context, code string, subtotal int) (DiscountCode, int, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	d, err := scanDiscount(s.db.QueryRow(ctx, `select `+discountCols+` from discount_codes where code = $1`, code))
	if err != nil {
		return d, 0, httpx.Unprocessable("invalid_discount", "That code is not valid.")
	}
	now := nowUTC()
	switch {
	case !d.Active, d.StartsAt != nil && now.Before(*d.StartsAt), d.EndsAt != nil && now.After(*d.EndsAt):
		return d, 0, httpx.Unprocessable("invalid_discount", "That code is not valid.")
	case d.MaxUses != nil && d.Uses >= *d.MaxUses:
		return d, 0, httpx.Unprocessable("discount_exhausted", "That code has been fully redeemed.")
	case subtotal < d.MinSubtotal:
		return d, 0, httpx.Unprocessable("discount_min", fmt.Sprintf("Spend at least £%d.%02d to use this code.", d.MinSubtotal/100, d.MinSubtotal%100))
	}
	amount := 0
	switch d.Type {
	case "percent":
		amount = (subtotal*d.Value + 50) / 100
	case "fixed":
		amount = min(d.Value, subtotal)
	case "free_delivery":
		amount = 0
	}
	return d, amount, nil
}

func (s *Store) ListDiscounts(ctx context.Context) ([]DiscountCode, error) {
	rows, err := s.db.Query(ctx, `select `+discountCols+` from discount_codes order by created_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DiscountCode{}
	for rows.Next() {
		d, err := scanDiscount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) UpsertDiscount(ctx context.Context, d DiscountCode) (DiscountCode, error) {
	d.Code = strings.ToUpper(strings.TrimSpace(d.Code))
	if d.Code == "" {
		return d, httpx.BadRequest("A code is required.")
	}
	if d.Type == "" {
		d.Type = "percent"
	}
	if d.Type == "percent" && (d.Value < 0 || d.Value > 100) {
		return d, httpx.BadRequest("Percent must be between 0 and 100.")
	}
	row := s.db.QueryRow(ctx, `insert into discount_codes (code, type, value, min_subtotal, max_uses, starts_at, ends_at, active)
		values ($1,$2,$3,$4,$5,$6,$7,$8) on conflict (code) do update set type=excluded.type, value=excluded.value, min_subtotal=excluded.min_subtotal,
		max_uses=excluded.max_uses, starts_at=excluded.starts_at, ends_at=excluded.ends_at, active=excluded.active returning `+discountCols,
		d.Code, d.Type, d.Value, d.MinSubtotal, d.MaxUses, d.StartsAt, d.EndsAt, d.Active)
	return scanDiscount(row)
}

func (s *Store) DeleteDiscount(ctx context.Context, code string) error {
	_, err := s.db.Exec(ctx, `delete from discount_codes where code = $1`, strings.ToUpper(code))
	return err
}
