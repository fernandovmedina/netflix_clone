package main

import (
	"net/http"
	"strconv"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
)

const rateLimitWindow = "1 minute"

func (app *application) rateLimited(action string, limit int, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var count int
		var retryAfter float64
		err := app.pool.QueryRow(r.Context(), `with expired as (
			select ctid from user_rate_limits
			where window_start<now()-interval '1 hour'
			order by window_start limit 100 for update skip locked
		), pruned as (
			delete from user_rate_limits buckets using expired
			where buckets.ctid=expired.ctid
		), counted as (
			insert into user_rate_limits(user_id,action,window_start,request_count)
			values($1,$2,date_bin($3::interval,now(),timestamptz 'epoch'),1)
			on conflict(user_id,action,window_start) do update set request_count=user_rate_limits.request_count+1
			returning request_count,window_start
		)
		select request_count,extract(epoch from (window_start+$3::interval-now())) from counted`, userID(r), action, rateLimitWindow).Scan(&count, &retryAfter)
		if err != nil {
			serverError(w, err)
			return
		}
		if count > limit {
			seconds := int(retryAfter) + 1
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			jsonx.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next(w, r)
	}
}
