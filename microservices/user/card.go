package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
)

type cardDetails struct {
	Number string `json:"number"`
	Expiry string `json:"exp"`
	CVV    string `json:"cvv"`
	Name   string `json:"name"`
}

func luhnValid(number string) bool {
	if len(number) < 13 || len(number) > 19 {
		return false
	}
	sum := 0
	double := false
	for index := len(number) - 1; index >= 0; index-- {
		digit := int(number[index] - '0')
		if digit < 0 || digit > 9 {
			return false
		}
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	return sum%10 == 0
}

func validExpiry(value string, now time.Time) bool {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 {
		return false
	}
	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return false
	}
	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	if len(parts[1]) == 2 {
		year += 2000
	} else if len(parts[1]) != 4 {
		return false
	}
	return year > now.Year() || (year == now.Year() && month >= int(now.Month()))
}

func cardBrand(number string) string {
	switch {
	case strings.HasPrefix(number, "4"):
		return "visa"
	case len(number) >= 2 && (number[:2] == "34" || number[:2] == "37"):
		return "amex"
	case len(number) >= 2 && number[:2] >= "51" && number[:2] <= "55":
		return "mastercard"
	case strings.HasPrefix(number, "6011") || strings.HasPrefix(number, "65"):
		return "discover"
	default:
		return "unknown"
	}
}

func validateCard(card cardDetails, now time.Time) (last4, brand string, ok bool) {
	if !luhnValid(card.Number) || !validExpiry(card.Expiry, now) || strings.TrimSpace(card.Name) == "" {
		return "", "", false
	}
	if len(card.CVV) < 3 || len(card.CVV) > 4 {
		return "", "", false
	}
	for _, char := range card.CVV {
		if !unicode.IsDigit(char) {
			return "", "", false
		}
	}
	return card.Number[len(card.Number)-4:], cardBrand(card.Number), true
}

func (app *application) cardPayment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PlanID int         `json:"plan_id"`
		Code   string      `json:"code"`
		Card   cardDetails `json:"card"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !validDatabaseID(in.PlanID) {
		jsonx.Error(w, http.StatusBadRequest, "valid plan_id is required")
		return
	}
	last4, brand, valid := validateCard(in.Card, time.Now())
	// Sensitive fields are discarded before any database or error path.
	in.Card.Number = ""
	in.Card.CVV = ""
	if !valid {
		jsonx.Error(w, http.StatusBadRequest, "invalid card details")
		return
	}
	out, err := app.createPayment(r.Context(), userID(r), paymentRequest{PlanID: in.PlanID, Code: in.Code}, paymentInsert{
		Method: "card", Status: "paid", Last4: last4, Brand: brand, Paid: true,
	})
	if err != nil {
		paymentError(w, err)
		return
	}
	jsonx.Write(w, http.StatusCreated, out)
}
