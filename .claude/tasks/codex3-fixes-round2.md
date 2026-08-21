# Codex 3 — Fixes from review round 2

## 1. `[CRITICAL]` Full card number and expiry persisted to localStorage

`frontend/app/signup/payment/card/page.tsx:46`

Confirmed in the **built bundle**, not just the source:

```
localStorage.setItem("signup_cardnumber",p),
localStorage.setItem("signup_cardexpiration",h),
localStorage.setItem("signup_cardname",x)
```

Any same-origin XSS, any malicious browser extension, or anyone with access to the machine can recover a full PAN indefinitely — `localStorage` has no expiry. This violates the explicit "never store a PAN" rule in ARCHITECTURE.md §10 and INTEGRATION.md's payment security list.

This is pre-existing code from the original signup flow, not something you introduced — but it ships in the bundle you built, so it is ours to fix.

### Fix

- Remove **every** `localStorage` write of card data: number, expiration, CVV, and cardholder name. Grep the whole frontend for `signup_card` and remove the persistence path entirely.
- Card details live in transient React state only, are submitted **once** to `POST /api/v1/payments/card`, and are never written to any browser storage, URL, query param, or analytics call.
- After submission, clear the fields from state.
- The only card data allowed to persist anywhere is what the **backend** returns: `card_last4` and `card_brand`. Render those from the payment response, never from client memory of the PAN.
- Also remove any pre-existing `signup_card*` values already sitting in users' `localStorage` — on mount of the payment flow, `localStorage.removeItem` the old keys so previously stored PANs are cleaned up rather than left behind forever.
- While you are in there: make sure the CVV is never logged, never put in a `console.log`, and not included in any error report.

### Verify

Build the bundle and grep it — the same way Codex 2 caught it:

```
pnpm build
rg -o 'localStorage\.setItem\("signup_card.{0,80}' .next/static/chunks/*.js
```

That must return nothing. Paste the empty result in your report.

Then walk the card flow in a browser with devtools open and confirm `localStorage` and `sessionStorage` hold no card data at any point.

---

## Notes

Codex 1 is implementing `POST /api/v1/payments/card` concurrently. Code against the contract in ARCHITECTURE.md §6: send `{plan_id, code?, card{number, exp, cvv, name}}` and never a price, subtotal or total. If the endpoint is not up yet, wire it correctly and degrade gracefully rather than blocking.

Do not commit.
