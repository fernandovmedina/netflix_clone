"use client";

import { DiscountField } from "@/components/payments/DiscountField";
import { PaymentBreakdown } from "@/components/payments/PaymentBreakdown";
import { CheckoutColumn, PaymentShell } from "@/components/payments/PaymentShell";
import { useCheckoutPlan } from "@/components/payments/useCheckoutPlan";
import type { DiscountPreview } from "@/utils/api/client";
import { formatCents } from "@/utils/payments";
import Link from "next/link";
import { useState } from "react";

export default function GiftCode() {
  const { plan, code, setCode, loading, error } = useCheckoutPlan();
  const [preview, setPreview] = useState<DiscountPreview | null>(null);
  return <PaymentShell><CheckoutColumn>
    <Link href="/signup/payment" className="text-blue-700 underline">← Change payment method</Link>
    <p className="mt-5 text-sm">STEP <strong>4</strong> OF <strong>4</strong></p><h1 className="mt-1 text-3xl font-black">Apply a gift code</h1>
    <p className="mt-3 text-gray-600">Gift Code is a discount. After validating it, complete payment by Card or OXXO.</p>
    {loading && <p className="mt-8 animate-pulse">Loading checkout…</p>}{error && <p role="alert" className="mt-5 rounded bg-red-50 p-4 text-red-700">{error}</p>}
    {plan && <>
      <div className="mt-6 flex items-center justify-between rounded bg-gray-100 p-4"><div><strong>{formatCents(plan.price, plan.currency)}/month</strong><p className="text-sm text-gray-600">{plan.name}</p></div><Link href="/signup/planform?change=true&paymentType=gift_code" className="font-bold text-blue-700 underline">Change</Link></div>
      <DiscountField plan={plan} code={code} onCodeChange={setCode} onPreview={setPreview} />
      {!preview?.valid && <PaymentBreakdown subtotal={plan.price} discount={0} total={plan.price} currency={plan.currency} />}
      {preview?.valid && <div className="mt-6 grid gap-3 sm:grid-cols-2"><Link href="/signup/payment/card" className="flex min-h-14 items-center justify-center rounded bg-red-600 px-5 font-bold text-white">Pay by Card</Link><Link href="/signup/payment/oxxo" className="flex min-h-14 items-center justify-center rounded border border-gray-800 px-5 font-bold">Pay by OXXO</Link></div>}
    </>}
  </CheckoutColumn></PaymentShell>;
}
