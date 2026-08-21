package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/authctx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMoneyConversionUsesExactCents(t *testing.T) {
	for input, want := range map[string]int64{"0": 0, "99.00": 9900, "149.5": 14950, "219.99": 21999} {
		got, err := parseCents(input)
		if err != nil || got != want {
			t.Errorf("parseCents(%q)=%d,%v want %d", input, got, err, want)
		}
		if roundTrip := decimalFromCents(want); roundTrip != fmt.Sprintf("%d.%02d", want/100, want%100) {
			t.Errorf("decimalFromCents(%d)=%s", want, roundTrip)
		}
	}
	if _, err := parseCents("10.001"); err == nil {
		t.Fatal("accepted fractional cents")
	}
}

func TestCardValidation(t *testing.T) {
	future := fmt.Sprintf("12/%d", time.Now().Year()+1)
	last4, brand, ok := validateCard(cardDetails{Number: "4111111111111111", Expiry: future, CVV: "123", Name: "Test User"}, time.Now())
	if !ok || last4 != "1111" || brand != "visa" {
		t.Fatalf("valid card rejected: last4=%q brand=%q ok=%v", last4, brand, ok)
	}
	for _, card := range []cardDetails{
		{Number: "4111111111111112", Expiry: future, CVV: "123", Name: "Test User"},
		{Number: "4111111111111111", Expiry: "01/2020", CVV: "123", Name: "Test User"},
		{Number: "4111111111111111", Expiry: future, CVV: "12x", Name: "Test User"},
	} {
		if _, _, ok = validateCard(card, time.Now()); ok {
			t.Fatalf("invalid card accepted: %#v", card)
		}
	}
}

func TestDiscountValidityRules(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	one := 1
	for name, discount := range map[string]discountRow{
		"inactive":       {Active: false, PerUserLimit: 1},
		"not started":    {Active: true, StartsAt: &future, PerUserLimit: 1},
		"expired":        {Active: true, ExpiresAt: &past, PerUserLimit: 1},
		"globally spent": {Active: true, MaxRedemptions: &one, RedemptionCount: 1, PerUserLimit: 1},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateDiscount(discount, 0, now); err == nil {
				t.Fatal("invalid discount accepted")
			}
		})
	}
	if err := validateDiscount(discountRow{Active: true, PerUserLimit: 1}, 1, now); err == nil {
		t.Fatal("per-user limit ignored")
	}
}

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("PHASE3_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("PHASE2_TEST_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("PHASE3_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func fixtureUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(), `insert into users(email,name,email_verified) values($1,'Test',true) returning id`, uuid.NewString()+"@example.test").Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func fixtureMovie(t *testing.T, pool *pgxpool.Pool) (titleID, movieID int) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), `insert into titles(type,title,published) values('Movie',$1,false) returning id_title`, "user-test-"+uuid.NewString()).Scan(&titleID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `insert into movies(id_title,duration) values($1,100) returning id_movie`, titleID).Scan(&movieID); err != nil {
		t.Fatal(err)
	}
	return titleID, movieID
}

func testHandler(app *application) http.Handler {
	mux := http.NewServeMux()
	app.routes(mux)
	return mux
}

func request(handler http.Handler, user uuid.UUID, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(authctx.UserIDHeader, user.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestWatchProgressParallelUpsertsConverge(t *testing.T) {
	pool := integrationPool(t)
	user := fixtureUser(t, pool)
	_, movie := fixtureMovie(t, pool)
	handler := testHandler(&application{pool: pool, simulationEnabled: true})
	const writers = 16
	statuses := make(chan int, writers)
	var wg sync.WaitGroup
	for index := 1; index <= writers; index++ {
		wg.Add(1)
		go func(value int) {
			defer wg.Done()
			rec := request(handler, user, http.MethodPut, fmt.Sprintf("/api/v1/progress/movie/%d", movie), fmt.Sprintf(`{"current_time_seconds":%d}`, value))
			statuses <- rec.Code
		}(index)
	}
	wg.Wait()
	close(statuses)
	for status := range statuses {
		if status != http.StatusOK {
			t.Errorf("parallel upsert status=%d", status)
		}
	}
	var count int
	if err := pool.QueryRow(context.Background(), `select count(*) from watch_progress where user_id=$1 and id_movie=$2`, user, movie).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("progress rows=%d want 1", count)
	}
}

func TestUserResourcesAreOwnerScoped(t *testing.T) {
	pool := integrationPool(t)
	owner, attacker := fixtureUser(t, pool), fixtureUser(t, pool)
	title, movie := fixtureMovie(t, pool)
	handler := testHandler(&application{pool: pool, simulationEnabled: true})

	profileRec := request(handler, owner, http.MethodPost, "/api/v1/profiles", `{"name":"Owner"}`)
	if profileRec.Code != http.StatusCreated {
		t.Fatalf("create profile status=%d body=%s", profileRec.Code, profileRec.Body.String())
	}
	var created profile
	if err := json.NewDecoder(profileRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		rec := request(handler, attacker, method, "/api/v1/profiles/"+created.ID.String(), "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("attacker %s profile status=%d body=%s", method, rec.Code, rec.Body.String())
		}
	}

	if rec := request(handler, owner, http.MethodPost, "/api/v1/favorites", fmt.Sprintf(`{"title_id":%d}`, title)); rec.Code != http.StatusCreated {
		t.Fatalf("favorite create status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := request(handler, attacker, http.MethodDelete, fmt.Sprintf("/api/v1/favorites/%d", title), ""); rec.Code != http.StatusNotFound {
		t.Errorf("attacker delete favorite status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := request(handler, owner, http.MethodPut, fmt.Sprintf("/api/v1/progress/movie/%d", movie), `{"current_time_seconds":30}`); rec.Code != http.StatusOK {
		t.Fatalf("progress create status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := request(handler, attacker, http.MethodGet, fmt.Sprintf("/api/v1/progress/movie/%d", movie), ""); rec.Code != http.StatusNotFound {
		t.Errorf("attacker read progress status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDiscountRedemptionIsSerialized(t *testing.T) {
	pool := integrationPool(t)
	users := []uuid.UUID{fixtureUser(t, pool), fixtureUser(t, pool)}
	code := "ONE-" + uuid.NewString()
	if _, err := pool.Exec(context.Background(), `insert into discounts(code,kind,value,max_redemptions,per_user_limit,active) values($1,'fixed',10.00,1,1,true)`, code); err != nil {
		t.Fatal(err)
	}
	var planID int
	if err := pool.QueryRow(context.Background(), `select id from plans where active=true order by id limit 1`).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	handler := testHandler(&application{pool: pool, simulationEnabled: true})
	preview := request(handler, users[0], http.MethodPost, "/api/v1/discounts/validate", fmt.Sprintf(`{"plan_id":%d,"code":%q}`, planID, code))
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", preview.Code, preview.Body.String())
	}
	var before int
	if err := pool.QueryRow(context.Background(), `select redemption_count from discounts where code=$1`, code).Scan(&before); err != nil || before != 0 {
		t.Fatalf("preview mutated redemption count: count=%d err=%v", before, err)
	}
	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for _, id := range users {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			body := fmt.Sprintf(`{"plan_id":%d,"code":%q,"total":1,"subtotal":1,"discount_amount":999999}`, planID, code)
			statuses <- request(handler, id, http.MethodPost, "/api/v1/payments/oxxo", body).Code
		}(id)
	}
	wg.Wait()
	close(statuses)
	got := []int{}
	for status := range statuses {
		got = append(got, status)
	}
	sort.Ints(got)
	if len(got) != 2 || got[0] != http.StatusCreated || got[1] != http.StatusConflict {
		t.Fatalf("parallel payment statuses=%v, want [201 409]", got)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `select redemption_count from discounts where code=$1`, code).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("redemption_count=%d want 1", count)
	}
	var subtotal, total, planPrice string
	if err := pool.QueryRow(context.Background(), `select p.subtotal::text,p.total::text,pl.price::text from payments p join plans pl on pl.id=p.plan_id where p.discount_id=(select id from discounts where code=$1)`, code).Scan(&subtotal, &total, &planPrice); err != nil {
		t.Fatal(err)
	}
	if subtotal != planPrice {
		t.Fatalf("client subtotal was trusted: payment=%s plan=%s", subtotal, planPrice)
	}
	wantTotal, _ := parseCents(planPrice)
	wantTotal -= 1000
	if total != decimalFromCents(wantTotal) {
		t.Fatalf("server total=%s want=%s", total, decimalFromCents(wantTotal))
	}
}

func TestCardPaymentCreatesPaidSubscriptionAndOnlyCardMetadata(t *testing.T) {
	pool := integrationPool(t)
	user := fixtureUser(t, pool)
	var planID int
	if err := pool.QueryRow(context.Background(), `select id from plans where active=true order by id limit 1`).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	handler := testHandler(&application{pool: pool, simulationEnabled: true})
	expiry := fmt.Sprintf("12/%d", time.Now().Year()+1)
	body := fmt.Sprintf(`{"plan_id":%d,"total":1,"card":{"number":"4111111111111111","exp":%q,"cvv":"123","name":"Test User"}}`, planID, expiry)
	rec := request(handler, user, http.MethodPost, "/api/v1/payments/card", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("card payment status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payment paymentResponse
	if err := json.NewDecoder(rec.Body).Decode(&payment); err != nil {
		t.Fatal(err)
	}
	var status, last4, brand string
	if err := pool.QueryRow(context.Background(), `select status::text,card_last4,card_brand from payments where id=$1 and user_id=$2`, payment.ID, user).Scan(&status, &last4, &brand); err != nil {
		t.Fatal(err)
	}
	if status != "paid" || last4 != "1111" || brand != "visa" {
		t.Fatalf("stored payment metadata status=%s last4=%s brand=%s", status, last4, brand)
	}
	if err := pool.QueryRow(context.Background(), `select status from subscriptions where user_id=$1 and plan_id=$2`, user, planID).Scan(&status); err != nil || status != "active" {
		t.Fatalf("subscription status=%q err=%v", status, err)
	}
}

func TestOXXOSimulationAndPaymentViewAreOwnerOnly(t *testing.T) {
	pool := integrationPool(t)
	owner, attacker := fixtureUser(t, pool), fixtureUser(t, pool)
	var planID int
	if err := pool.QueryRow(context.Background(), `select id from plans where active=true order by id limit 1`).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	handler := testHandler(&application{pool: pool, simulationEnabled: true})
	rec := request(handler, owner, http.MethodPost, "/api/v1/payments/oxxo", fmt.Sprintf(`{"plan_id":%d}`, planID))
	if rec.Code != http.StatusCreated {
		t.Fatalf("oxxo status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payment paymentResponse
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&payment); err != nil || payment.Reference == nil {
		t.Fatalf("decode payment: %#v err=%v", payment, err)
	}
	if got := request(handler, attacker, http.MethodPost, "/api/v1/payments/oxxo/"+*payment.Reference+"/simulate-payment", ""); got.Code != http.StatusNotFound {
		t.Errorf("attacker simulation status=%d body=%s", got.Code, got.Body.String())
	}
	if got := request(handler, attacker, http.MethodGet, "/api/v1/payments/"+payment.ID.String(), ""); got.Code != http.StatusNotFound {
		t.Errorf("attacker payment view status=%d body=%s", got.Code, got.Body.String())
	}
	if got := request(handler, owner, http.MethodPost, "/api/v1/payments/oxxo/"+*payment.Reference+"/simulate-payment", ""); got.Code != http.StatusOK {
		t.Errorf("owner simulation status=%d body=%s", got.Code, got.Body.String())
	}
}
