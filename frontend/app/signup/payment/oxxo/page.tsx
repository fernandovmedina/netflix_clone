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
import { useState } from "react";

function Barcode({ reference }: { reference: string }) {
  const bars = reference.split("").flatMap((digit, index) => [Number(digit) % 4 + 1, (index * 3 + Number(digit)) % 3 + 1]);
  return <div aria-label={`Barcode reference ${reference}`} className="flex h-24 items-stretch justify-center overflow-hidden bg-white px-3 py-2">{bars.map((width, index) => <span key={index} className={index % 2 === 0 ? "bg-black" : "bg-white"} style={{ width: `${width}px` }} />)}</div>;
}

export default function Oxxo() {
  const router = useRouter();
  const { plan, code, setCode, loading, error: planError } = useCheckoutPlan();
  const [preview, setPreview] = useState<DiscountPreview | null>(null);
  const [payment, setPayment] = useState<Payment | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const createVoucher = async () => {
    if (!plan) return;
    setSubmitting(true); setError("");
    try { setPayment(await paymentApi.createOxxo(plan.id, code.trim() || undefined)); }
    catch (reason) { setError(paymentErrorMessage(reason)); }
    finally { setSubmitting(false); }
  };

  const simulate = async () => {
    if (!payment?.reference) return;
    setSubmitting(true); setError("");
    try { const paid = await paymentApi.simulateOxxo(payment.reference); setPayment(paid); clearCheckoutState(); }
    catch (reason) { setError(paymentErrorMessage(reason)); }
    finally { setSubmitting(false); }
  };

  if (payment) return <PaymentShell><CheckoutColumn>
    <div className="overflow-hidden rounded-xl border-2 border-red-600 bg-white shadow-xl"><div className="flex items-center justify-between bg-red-600 p-5 text-white"><div><p className="text-xs font-bold uppercase tracking-widest">OXXO PAY voucher</p><h1 className="mt-1 text-2xl font-black">{payment.status === "paid" ? "Payment completed" : "Pay at any OXXO store"}</h1></div><Image src="/payments/oxxo.png" alt="OXXO" width={70} height={42} className="h-auto w-16 rounded bg-white p-1" /></div>
      <div className="p-5 sm:p-7"><Barcode reference={payment.reference ?? ""} /><p className="mt-2 break-all text-center font-mono text-lg tracking-widest">{payment.reference}</p><PaymentBreakdown subtotal={payment.subtotal} discount={payment.discount_amount} total={payment.total} currency={payment.currency} authoritative /><dl className="mt-5 grid gap-2 text-sm"><div className="flex justify-between gap-4"><dt>Status</dt><dd className="font-bold uppercase">{payment.status}</dd></div><div className="flex justify-between gap-4"><dt>Expires</dt><dd className="text-right">{payment.expires_at ? new Date(payment.expires_at).toLocaleString() : "—"}</dd></div></dl>
        {payment.status === "pending" ? <div className="mt-6 rounded border-2 border-dashed border-amber-500 bg-amber-50 p-4"><p className="font-black uppercase text-amber-900">Development simulation only</p><p className="mt-1 text-sm text-amber-800">This button simulates an OXXO store callback locally. It is not a real payment confirmation.</p><button type="button" onClick={simulate} disabled={submitting} className="mt-4 min-h-12 w-full rounded bg-amber-600 px-5 font-bold text-white disabled:opacity-50">{submitting ? "Simulating…" : "Simulate payment (dev only)"}</button></div> : <button type="button" onClick={() => router.push("/home/browse")} className="mt-6 min-h-12 w-full rounded bg-red-600 px-5 font-bold text-white">Enter Netflix</button>}
        {error && <p role="alert" className="mt-4 text-red-700">{error}</p>}
      </div></div>
  </CheckoutColumn></PaymentShell>;

  return <PaymentShell><CheckoutColumn>
    <Link href="/signup/payment" className="text-blue-700 underline">← Change payment method</Link><p className="mt-5 text-sm">STEP <strong>4</strong> OF <strong>4</strong></p><h1 className="mt-1 text-3xl font-black">Create an OXXO voucher</h1><p className="mt-3 text-gray-600">The server will generate a barcode reference, final amount, and expiry.</p>
    {loading && <p className="mt-8 animate-pulse">Loading checkout…</p>}{planError && <p role="alert" className="mt-5 rounded bg-red-50 p-4 text-red-700">{planError}</p>}
    {plan && <><div className="mt-6 flex items-center justify-between rounded bg-gray-100 p-4"><div><strong>{formatCents(plan.price, plan.currency)}/month</strong><p className="text-sm text-gray-600">{plan.name}</p></div><Link href="/signup/planform?change=true&paymentType=oxxo" className="font-bold text-blue-700 underline">Change</Link></div><DiscountField plan={plan} code={code} onCodeChange={setCode} onPreview={setPreview} />{!preview?.valid && <PaymentBreakdown subtotal={plan.price} discount={0} total={plan.price} currency={plan.currency} />}{error && <p role="alert" className="mt-4 rounded bg-red-50 p-3 text-red-700">{error}</p>}<button type="button" onClick={createVoucher} disabled={submitting} className="my-5 min-h-14 w-full rounded bg-red-600 px-5 text-xl font-bold text-white disabled:opacity-50">{submitting ? "Creating voucher…" : "Get OXXO reference"}</button></>}
  </CheckoutColumn></PaymentShell>;
}
