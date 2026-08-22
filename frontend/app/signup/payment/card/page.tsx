"use client";

import { DiscountField } from "@/components/payments/DiscountField";
import { PaymentBreakdown } from "@/components/payments/PaymentBreakdown";
import { CheckoutColumn, PaymentShell } from "@/components/payments/PaymentShell";
import { useCheckoutPlan } from "@/components/payments/useCheckoutPlan";
import { paymentApi, type DiscountPreview, type Payment } from "@/utils/api/client";
import { clearCheckoutState, formatCents, paymentErrorMessage } from "@/utils/payments";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const LEGACY_CARD_KEYS = ["signup_cardnumber", "signup_cardexpiration", "signup_cardname", "signup_cardcvv"];

export default function Card() {
  const router = useRouter();
  const { plan, code, setCode, loading, error: planError } = useCheckoutPlan();
  const [preview, setPreview] = useState<DiscountPreview | null>(null);
  const [cardNumber, setCardNumber] = useState("");
  const [cardExpiration, setCardExpiration] = useState("");
  const [cvv, setCvv] = useState("");
  const [cardName, setCardName] = useState("");
  const [agreed, setAgreed] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [payment, setPayment] = useState<Payment | null>(null);

  useEffect(() => { LEGACY_CARD_KEYS.forEach((key) => localStorage.removeItem(key)); }, []);

  const clearCardState = () => {
    setCardNumber(""); setCardExpiration(""); setCvv(""); setCardName("");
  };

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!plan || !agreed) { setError("Agree to the terms before starting membership."); return; }
    setSubmitting(true); setError("");
    try {
      const result = await paymentApi.payByCard({
        plan_id: plan.id,
        ...(code.trim() ? { code: code.trim() } : {}),
        card: { number: cardNumber.replace(/\D/g, ""), exp: cardExpiration, cvv, name: cardName.trim() },
      });
      clearCardState();
      clearCheckoutState();
      setPayment(result);
    } catch (reason) {
      clearCardState();
      setError(paymentErrorMessage(reason));
    } finally { setSubmitting(false); }
  };

  if (payment) return <PaymentShell><CheckoutColumn>
    <div className="rounded-xl border border-emerald-300 bg-emerald-50 p-6"><p className="text-sm font-bold uppercase tracking-wider text-emerald-700">Payment successful</p><h1 className="mt-2 text-3xl font-black">Membership active</h1><p className="mt-3">Paid with {payment.card_brand ?? "card"} ending in {payment.card_last4 ?? "••••"}.</p><PaymentBreakdown subtotal={payment.subtotal} discount={payment.discount_amount} total={payment.total} currency={payment.currency} authoritative /><button type="button" onClick={() => router.push("/home/browse")} className="mt-6 min-h-12 w-full rounded bg-red-600 px-5 font-bold text-white">Enter Netflix</button></div>
  </CheckoutColumn></PaymentShell>;

  return <PaymentShell><CheckoutColumn>
    <Link href="/signup/payment" className="text-blue-700 underline">← Change payment method</Link>
    <p className="mt-5 text-sm">STEP <strong>4</strong> OF <strong>4</strong></p><h1 className="mt-1 text-3xl font-black">Set up your credit or debit card.</h1>
    <div className="mt-4 flex gap-1">{["visa", "mastercard", "amex", "carnet"].map((brand) => <Image key={brand} src={`/payments/${brand}.png`} alt={brand} width={45} height={28} className="h-7 w-auto object-contain" />)}</div>
    {loading && <p className="mt-8 animate-pulse">Loading checkout…</p>}{planError && <p role="alert" className="mt-5 rounded bg-red-50 p-4 text-red-700">{planError}</p>}
    {plan && <form onSubmit={submit} className="mt-5">
      <input required inputMode="numeric" autoComplete="cc-number" value={cardNumber} onChange={(event) => { const digits = event.target.value.replace(/\D/g, "").slice(0, 19); setCardNumber(digits.replace(/(.{4})/g, "$1 ").trim()); }} placeholder="Card number" className="min-h-14 w-full rounded border border-gray-400 px-4 text-base" />
      <div className="mt-3 grid grid-cols-2 gap-3"><input required inputMode="numeric" autoComplete="cc-exp" value={cardExpiration} onChange={(event) => { const value = event.target.value.replace(/\D/g, "").slice(0, 4); setCardExpiration(value.length > 2 ? `${value.slice(0, 2)}/${value.slice(2)}` : value); }} placeholder="MM/YY" className="min-h-14 min-w-0 rounded border border-gray-400 px-4 text-base" /><input required inputMode="numeric" autoComplete="cc-csc" value={cvv} onChange={(event) => setCvv(event.target.value.replace(/\D/g, "").slice(0, 4))} placeholder="CVV" type="password" className="min-h-14 min-w-0 rounded border border-gray-400 px-4 text-base" /></div>
      <input required autoComplete="cc-name" value={cardName} onChange={(event) => setCardName(event.target.value)} placeholder="Name on card" className="mt-3 min-h-14 w-full rounded border border-gray-400 px-4 text-base" />
      <div className="mt-5 flex items-center justify-between rounded bg-gray-100 p-4"><div><strong>{formatCents(plan.price, plan.currency)}/month</strong><p className="text-sm text-gray-600">{plan.name}</p></div><Link href="/signup/planform?change=true&paymentType=card" className="font-bold text-blue-700 underline">Change</Link></div>
      <DiscountField plan={plan} code={code} onCodeChange={setCode} onPreview={setPreview} />
      {!preview?.valid && <PaymentBreakdown subtotal={plan.price} discount={0} total={plan.price} currency={plan.currency} />}
      {error && <p role="alert" className="mt-4 rounded bg-red-50 p-3 text-red-700">{error}</p>}
      <label className="mt-5 flex min-h-11 items-start gap-3 text-sm"><input type="checkbox" checked={agreed} onChange={(event) => setAgreed(event.target.checked)} className="mt-1 h-5 w-5 shrink-0" /><span>I agree to the Terms of Use and recurring monthly membership until cancellation.</span></label>
      <button disabled={submitting} className="my-5 min-h-14 w-full rounded bg-red-600 px-5 text-xl font-bold text-white hover:bg-red-700 disabled:opacity-50">{submitting ? "Confirming secure total…" : "Start Membership"}</button>
      <p className="text-xs text-gray-500">Card details are sent once to the payment server and are never stored in browser storage. Only the returned card brand and last four digits are displayed.</p>
    </form>}
  </CheckoutColumn></PaymentShell>;
}
