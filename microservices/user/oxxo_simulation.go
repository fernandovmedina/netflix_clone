package main

// This file is intentionally isolated: it is a local-development simulation
// of an OXXO store callback, not a payment-provider integration.

import (
	"errors"
	"net/http"
	"time"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
	"github.com/jackc/pgx/v5"
)

var errPaymentExpired = errors.New("payment reference has expired")

func (app *application) simulateOXXOPayment(w http.ResponseWriter, r *http.Request) {
	if !app.simulationEnabled {
		jsonx.Error(w, http.StatusNotFound, "payment simulation is disabled")
		return
	}
	reference := r.PathValue("ref")
	if reference == "" {
		jsonx.Error(w, http.StatusBadRequest, "reference is required")
		return
	}
	tx, err := app.pool.Begin(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var paymentID string
	var planID int
	var status string
	var expires time.Time
	err = tx.QueryRow(r.Context(), `select id::text,plan_id,status::text,expires_at from payments
		where reference=$1 and user_id=$2 and method='oxxo' for update`, reference, userID(r)).Scan(&paymentID, &planID, &status, &expires)
	if err == pgx.ErrNoRows {
		jsonx.Error(w, http.StatusNotFound, "payment reference not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if status != "pending" {
		jsonx.Error(w, http.StatusConflict, "payment is not pending")
		return
	}
	if !time.Now().Before(expires) {
		if _, err = tx.Exec(r.Context(), `update payments set status='expired',updated_at=now() where id=$1`, paymentID); err != nil {
			serverError(w, err)
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			serverError(w, err)
			return
		}
		jsonx.Error(w, http.StatusConflict, errPaymentExpired.Error())
		return
	}
	if _, err = tx.Exec(r.Context(), `update payments set status='paid',paid_at=now(),updated_at=now() where id=$1`, paymentID); err == nil {
		err = activateSubscription(r.Context(), tx, userID(r), planID)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		serverError(w, err)
		return
	}
	out, err := scanPayment(app.pool.QueryRow(r.Context(), `select id,plan_id,method::text,status::text,subtotal::text,discount_amount::text,total::text,currency,
		reference,card_last4,card_brand,expires_at,paid_at,simulated,created_at from payments where id=$1 and user_id=$2`, paymentID, userID(r)))
	if err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, http.StatusOK, out)
}
