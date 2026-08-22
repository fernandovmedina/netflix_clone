# Codex 3 — Phase 4 (M14 payments UI + M15 responsive)

Repo root: `/Users/froot/Documents/workspace/fernandovmedina/web/netflix_clone`
Scope: `frontend/**` only.

Do Part A first — it is a CRITICAL security finding.

---

## Part A — CRITICAL fix (do first)

Execute `.claude/tasks/codex3-fixes-round2.md`: the card page persists the full PAN and expiry to `localStorage`, confirmed in the built bundle. Remove it, clear any previously stored values, and prove it with a grep of the rebuilt bundle.

---

## Part B — M14: payments wired to backend totals

Codex 1 has now shipped the payments API. Endpoints (ARCHITECTURE.md §6):

```
GET  /api/v1/plans
POST /api/v1/discounts/validate            {code, plan_id} -> {valid, discount_amount, total}
POST /api/v1/payments/card                 {plan_id, code?, card{...}}
POST /api/v1/payments/oxxo                 {plan_id, code?} -> {reference, amount, expires_at}
POST /api/v1/payments/oxxo/:ref/simulate-payment
GET  /api/v1/payments/:id
```

- `app/signup/planform/page.tsx` currently hardcodes plans. Load them from `GET /api/v1/plans` instead — price, name, quality and max streams all come from the backend.
- **The client never sends money.** Send `plan_id` and an optional `code`. Never send a price, subtotal, discount amount or total, even as a display value the server might read back. Codex 2 will attempt to manipulate these.
- Gift Code is the discount system (a decision already made — it is not a separate payment rail and it is not crypto). Wire the existing `app/signup/payment/gift_code/page.tsx` and a code field on the card and OXXO flows to `POST /api/v1/discounts/validate` for a **live preview**, and make clear in the UI that the final amount is confirmed by the server. The preview is advisory; render the authoritative total from the payment response.
- Show the discount breakdown: subtotal, discount applied, total — all from server numbers.
- Handle the real failure cases with useful messages rather than a generic error: invalid code, expired code, already redeemed by this user, code exhausted, card declined, and the 409 that comes back when two attempts race for a single-use code.
- **OXXO**: after creating a payment, render a realistic voucher — barcode reference, amount, expiry. Then, because this is a local simulation, offer a clearly-labelled dev-only "Simulate payment" action that calls the simulate endpoint and flips the UI to paid. Label it unmistakably as simulation; do not dress it up as a real payment confirmation.
- Money arrives as integer cents. Format for display; never do float arithmetic on it in JS.
- On success, land the user in the app with an active subscription.

There is no crypto payment method in this codebase and there never was — do not add one in order to remove it. If you touch `app/signup/payment/page.tsx`, leave Card / OXXO / Gift Code as the three methods.

## Part C — M15: responsive pass

The site must work at **375, 768, 1024 and 1440** px. Do not assume — resize and look at each one.

Known offenders found during review:
- The title modal is a fixed `w-[900px]` — it overflows every phone and most tablets.
- The payment and signup columns use `w-[30%]`, which collapses to an unusable sliver on mobile and strands content on wide screens.
- The home hero uses `pt-80 pl-20` with a fixed `w-[400px]` copy block.
- Carousels use `min-w-60` cards with `px-14` arrows; on a 375 px viewport the arrows overlap the cards.

Cover every page in INTEGRATION.md's list: navigation, auth, home/catalog, movie and series pages, the video player, payments, admin dashboard, admin uploads, forms and modals.

Specifics that matter:
- Navbar collapses to a working mobile menu — not a squashed desktop bar.
- Modals become full-screen sheets on small viewports and must be scrollable and dismissible.
- The player's controls must be touch-sized and must not depend on hover. The quality menu needs to be reachable by tap.
- Admin tables must not force horizontal page scroll — let the table scroll inside its own container.
- Carousels: touch-swipe on mobile; hide or reposition the arrow buttons where they would overlap content.
- Forms: inputs at least 16 px on iOS or Safari zooms on focus.
- No horizontal page scroll at any of the four widths. That is the single easiest regression to check and the most common failure.

## Definition of done

- `pnpm build`, `pnpm lint`, `tsc --noEmit` all clean.
- The bundle grep for stored card data returns nothing.
- You have checked all four widths on every listed page and can say what you changed for each.
- The full payment flow works end to end against the running stack: pick a plan, apply a discount code, pay by card, and separately create and simulate an OXXO payment.

Report what you changed, what you verified, and anything still broken. Do not commit.

Work autonomously; do not stop to ask for confirmation.
