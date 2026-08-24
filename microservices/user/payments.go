package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	errPlanNotFound       = errors.New("plan not found")
	errDiscountNotFound   = errors.New("discount not found")
	errDiscountInactive   = errors.New("discount is inactive")
	errDiscountNotStarted = errors.New("discount has not started")
	errDiscountExpired    = errors.New("discount has expired")
	errDiscountDefinition = errors.New("discount definition is invalid")
	errDiscountSpent      = errors.New("discount redemption limit reached")
)

type planResponse struct {
	ID         int    `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price"`
	Currency   string `json:"currency"`
	Quality    string `json:"quality"`
	MaxStreams int    `json:"max_streams"`
}

func (app *application) listPlans(w http.ResponseWriter, r *http.Request) {
	rows, err := app.pool.Query(r.Context(), `select id,code,name,price::text,currency,quality,max_streams from plans where active=true order by price,id`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	plans := []planResponse{}
	for rows.Next() {
		var item planResponse
		var price string
		if err = rows.Scan(&item.ID, &item.Code, &item.Name, &price, &item.Currency, &item.Quality, &item.MaxStreams); err != nil {
			serverError(w, err)
			return
		}
		item.PriceCents, err = parseCents(price)
		if err != nil {
			serverError(w, err)
			return
		}
		plans = append(plans, item)
	}
	jsonx.Write(w, http.StatusOK, plans)
}

type discountRow struct {
	ID              int
	Kind            string
	ValueHundredths int64
	MaxRedemptions  *int
	RedemptionCount int
	PerUserLimit    int
	StartsAt        *time.Time
	ExpiresAt       *time.Time
	Active          bool
}

type price struct {
	PlanID         int
	SubtotalCents  int64
	DiscountID     *int
	DiscountCents  int64
	TotalCents     int64
	Currency       string
	DiscountRecord *discountRow
}

func calculateDiscount(subtotal int64, discount discountRow) int64 {
	amount := discount.ValueHundredths
	if discount.Kind == "percent" {
		amount = subtotal * discount.ValueHundredths / 10000
	}
	if amount < 0 {
		amount = 0
	}
	if amount > subtotal {
		amount = subtotal
	}
	return amount
}

func validateDiscount(discount discountRow, userRedemptions int, now time.Time) error {
	if discount.ValueHundredths < 0 || discount.Kind == "percent" && discount.ValueHundredths > 10000 {
		return errDiscountDefinition
	}
	if !discount.Active {
		return errDiscountInactive
	}
	if discount.StartsAt != nil && now.Before(*discount.StartsAt) {
		return errDiscountNotStarted
	}
	if discount.ExpiresAt != nil && !now.Before(*discount.ExpiresAt) {
		return errDiscountExpired
	}
	if (discount.MaxRedemptions != nil && discount.RedemptionCount >= *discount.MaxRedemptions) || userRedemptions >= discount.PerUserLimit {
		return errDiscountSpent
	}
	return nil
}

func scanDiscount(row pgx.Row) (discountRow, error) {
	var discount discountRow
	var value string
	err := row.Scan(&discount.ID, &discount.Kind, &value, &discount.MaxRedemptions, &discount.RedemptionCount,
		&discount.PerUserLimit, &discount.StartsAt, &discount.ExpiresAt, &discount.Active)
	if err != nil {
		return discount, err
	}
	discount.ValueHundredths, err = parseCents(value)
	return discount, err
}

func (app *application) previewDiscount(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code   string `json:"code"`
		PlanID int    `json:"plan_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !validDatabaseID(in.PlanID) {
		jsonx.Error(w, http.StatusBadRequest, "valid plan_id is required")
		return
	}
	var subtotalText, currency string
	err := app.pool.QueryRow(r.Context(), `select price::text,currency from plans where id=$1 and active=true`, in.PlanID).Scan(&subtotalText, &currency)
	if err == pgx.ErrNoRows {
		jsonx.Error(w, http.StatusNotFound, "plan not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	subtotal, err := parseCents(subtotalText)
	if err != nil {
		serverError(w, err)
		return
	}
	result := map[string]any{"valid": false, "subtotal": subtotal, "discount_amount": int64(0), "total": subtotal, "currency": currency}
	discount, err := scanDiscount(app.pool.QueryRow(r.Context(), `select id,kind::text,value::text,max_redemptions,redemption_count,per_user_limit,starts_at,expires_at,active from discounts where code=$1`, strings.TrimSpace(in.Code)))
	if err == pgx.ErrNoRows {
		result["reason"] = "discount not found"
		jsonx.Write(w, http.StatusOK, result)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	var userUses int
	if err = app.pool.QueryRow(r.Context(), `select count(*) from discount_redemptions where discount_id=$1 and user_id=$2`, discount.ID, userID(r)).Scan(&userUses); err != nil {
		serverError(w, err)
		return
	}
	if err = validateDiscount(discount, userUses, time.Now()); err != nil {
		result["reason"] = err.Error()
		jsonx.Write(w, http.StatusOK, result)
		return
	}
	amount := calculateDiscount(subtotal, discount)
	result["valid"] = true
	result["discount_amount"] = amount
	result["total"] = subtotal - amount
	jsonx.Write(w, http.StatusOK, result)
}

type paymentRequest struct {
	PlanID int    `json:"plan_id"`
	Code   string `json:"code"`
}

type paymentResponse struct {
	ID                  uuid.UUID  `json:"id"`
	PlanID              int        `json:"plan_id"`
	Method              string     `json:"method"`
	Status              string     `json:"status"`
	SubtotalCents       int64      `json:"subtotal"`
	DiscountAmountCents int64      `json:"discount_amount"`
	TotalCents          int64      `json:"total"`
	AmountCents         int64      `json:"amount"`
	Currency            string     `json:"currency"`
	Reference           *string    `json:"reference,omitempty"`
	CardLast4           *string    `json:"card_last4,omitempty"`
	CardBrand           *string    `json:"card_brand,omitempty"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
	Simulated           bool       `json:"simulated"`
	CreatedAt           time.Time  `json:"created_at"`
}

func (app *application) priceForPayment(ctx context.Context, tx pgx.Tx, user uuid.UUID, planID int, code string) (price, error) {
	var out price
	out.PlanID = planID
	var subtotal string
	err := tx.QueryRow(ctx, `select price::text,currency from plans where id=$1 and active=true`, planID).Scan(&subtotal, &out.Currency)
	if err == pgx.ErrNoRows {
		return out, errPlanNotFound
	}
	if err != nil {
		return out, err
	}
	out.SubtotalCents, err = parseCents(subtotal)
	if err != nil {
		return out, err
	}
	out.TotalCents = out.SubtotalCents
	if strings.TrimSpace(code) == "" {
		return out, nil
	}
	discount, err := scanDiscount(tx.QueryRow(ctx, `select id,kind::text,value::text,max_redemptions,redemption_count,per_user_limit,starts_at,expires_at,active from discounts where code=$1 for update`, strings.TrimSpace(code)))
	if err == pgx.ErrNoRows {
		return out, errDiscountNotFound
	}
	if err != nil {
		return out, err
	}
	var userUses int
	if err = tx.QueryRow(ctx, `select count(*) from discount_redemptions where discount_id=$1 and user_id=$2`, discount.ID, user).Scan(&userUses); err != nil {
		return out, err
	}
	if err = validateDiscount(discount, userUses, time.Now()); err != nil {
		return out, err
	}
	out.DiscountRecord = &discount
	out.DiscountID = &discount.ID
	out.DiscountCents = calculateDiscount(out.SubtotalCents, discount)
	out.TotalCents = out.SubtotalCents - out.DiscountCents
	if out.TotalCents < 0 {
		out.TotalCents = 0
	}
	return out, nil
}

type paymentInsert struct {
	Method, Status, Last4, Brand string
	Reference                    *string
	ExpiresAt                    *time.Time
	Paid                         bool
}

func (app *application) createPayment(ctx context.Context, user uuid.UUID, request paymentRequest, insert paymentInsert) (paymentResponse, error) {
	tx, err := app.pool.Begin(ctx)
	if err != nil {
		return paymentResponse{}, err
	}
	defer tx.Rollback(ctx)
	computed, err := app.priceForPayment(ctx, tx, user, request.PlanID, request.Code)
	if err != nil {
		return paymentResponse{}, err
	}
	var out paymentResponse
	var last4, brand any
	if insert.Last4 != "" {
		last4, brand = insert.Last4, insert.Brand
	}
	err = tx.QueryRow(ctx, `insert into payments(user_id,plan_id,method,status,subtotal,discount_id,discount_amount,total,currency,reference,card_last4,card_brand,expires_at,paid_at,simulated)
		values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,case when $14 then now() end,true)
		returning id,created_at,paid_at`, user, request.PlanID, insert.Method, insert.Status, decimalFromCents(computed.SubtotalCents), computed.DiscountID,
		decimalFromCents(computed.DiscountCents), decimalFromCents(computed.TotalCents), computed.Currency, insert.Reference, last4, brand, insert.ExpiresAt, insert.Paid).Scan(&out.ID, &out.CreatedAt, &out.PaidAt)
	if err != nil {
		return out, err
	}
	if computed.DiscountRecord != nil {
		if _, err = tx.Exec(ctx, `update discounts set redemption_count=redemption_count+1 where id=$1`, computed.DiscountRecord.ID); err != nil {
			return out, err
		}
		if _, err = tx.Exec(ctx, `insert into discount_redemptions(discount_id,user_id,payment_id) values($1,$2,$3)`, computed.DiscountRecord.ID, user, out.ID); err != nil {
			return out, err
		}
	}
	if insert.Paid {
		if err = activateSubscription(ctx, tx, user, request.PlanID); err != nil {
			return out, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	out.PlanID, out.Method, out.Status = request.PlanID, insert.Method, insert.Status
	out.SubtotalCents, out.DiscountAmountCents, out.TotalCents, out.Currency = computed.SubtotalCents, computed.DiscountCents, computed.TotalCents, computed.Currency
	out.AmountCents = out.TotalCents
	out.Reference, out.ExpiresAt = insert.Reference, insert.ExpiresAt
	if insert.Last4 != "" {
		out.CardLast4, out.CardBrand = &insert.Last4, &insert.Brand
	}
	out.Simulated = true
	return out, nil
}

func activateSubscription(ctx context.Context, tx pgx.Tx, user uuid.UUID, planID int) error {
	_, err := tx.Exec(ctx, `insert into subscriptions(user_id,plan_id,status,current_period_end) values($1,$2,'active',now()+interval '1 month')
		on conflict(user_id) do update set plan_id=excluded.plan_id,status='active',current_period_end=excluded.current_period_end,updated_at=now()`, user, planID)
	return err
}

func paymentError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, errPlanNotFound):
		jsonx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errDiscountNotFound), errors.Is(err, errDiscountInactive), errors.Is(err, errDiscountNotStarted), errors.Is(err, errDiscountExpired), errors.Is(err, errDiscountDefinition):
		jsonx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errDiscountSpent), errors.As(err, &pgErr) && pgErr.Code == "23505":
		jsonx.Error(w, http.StatusConflict, "discount redemption limit reached")
	default:
		serverError(w, err)
	}
}

func newOXXOReference() (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	var digits strings.Builder
	for _, value := range raw {
		fmt.Fprintf(&digits, "%02d", int(value)%100)
	}
	return digits.String(), nil
}

func (app *application) oxxoPayment(w http.ResponseWriter, r *http.Request) {
	var in paymentRequest
	if !decode(w, r, &in) {
		return
	}
	if !validDatabaseID(in.PlanID) {
		jsonx.Error(w, http.StatusBadRequest, "valid plan_id is required")
		return
	}
	reference, err := newOXXOReference()
	if err != nil {
		serverError(w, err)
		return
	}
	expires := time.Now().UTC().Add(72 * time.Hour)
	out, err := app.createPayment(r.Context(), userID(r), in, paymentInsert{Method: "oxxo", Status: "pending", Reference: &reference, ExpiresAt: &expires})
	if err != nil {
		paymentError(w, err)
		return
	}
	jsonx.Write(w, http.StatusCreated, out)
}

func scanPayment(row pgx.Row) (paymentResponse, error) {
	var out paymentResponse
	var subtotal, discount, total string
	err := row.Scan(&out.ID, &out.PlanID, &out.Method, &out.Status, &subtotal, &discount, &total, &out.Currency,
		&out.Reference, &out.CardLast4, &out.CardBrand, &out.ExpiresAt, &out.PaidAt, &out.Simulated, &out.CreatedAt)
	if err != nil {
		return out, err
	}
	if out.SubtotalCents, err = parseCents(subtotal); err == nil {
		out.DiscountAmountCents, err = parseCents(discount)
	}
	if err == nil {
		out.TotalCents, err = parseCents(total)
		out.AmountCents = out.TotalCents
	}
	return out, err
}

func (app *application) getPayment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonx.Error(w, http.StatusBadRequest, "invalid payment id")
		return
	}
	out, err := scanPayment(app.pool.QueryRow(r.Context(), `select id,plan_id,method::text,status::text,subtotal::text,discount_amount::text,total::text,currency,
		reference,card_last4,card_brand,expires_at,paid_at,simulated,created_at from payments where id=$1 and user_id=$2`, id, userID(r)))
	if err == pgx.ErrNoRows {
		jsonx.Error(w, http.StatusNotFound, "payment not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, http.StatusOK, out)
}
