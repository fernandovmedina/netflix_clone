"use client";

import { CheckoutColumn, PaymentShell } from "@/components/payments/PaymentShell";
import { paymentApi, type Plan } from "@/utils/api/client";
import { formatCents, rememberPlan, selectedPlanId } from "@/utils/payments";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function Planform() {
  const router = useRouter();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [selected, setSelected] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [returnTo, setReturnTo] = useState("/signup/payment");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const paymentType = params.get("paymentType");
    if (params.get("change") && paymentType && ["card", "oxxo", "gift_code"].includes(paymentType)) setReturnTo(`/signup/payment/${paymentType}`);
    paymentApi.plans().then((items) => {
      setPlans(items);
      const remembered = selectedPlanId();
      setSelected(items.some((plan) => plan.id === remembered) ? remembered : items[0]?.id ?? null);
    }).catch((reason: unknown) => setError(reason instanceof Error ? reason.message : "Unable to load plans.")).finally(() => setLoading(false));
  }, []);

  const continueToPayment = () => {
    const plan = plans.find((item) => item.id === selected);
    if (!plan) return;
    rememberPlan(plan);
    router.push(returnTo);
  };

  return <PaymentShell><CheckoutColumn wide>
    <p className="text-sm">STEP <strong>3</strong> OF <strong>4</strong></p>
    <h1 className="mt-2 text-3xl font-black sm:text-4xl">Choose the plan that&apos;s right for you</h1>
    <p className="mt-3 text-gray-600">Plan availability and pricing come directly from the subscription server.</p>
    {loading && <p className="mt-10 animate-pulse">Loading plans…</p>}
    {error && <p role="alert" className="mt-8 rounded bg-red-50 p-4 text-red-700">{error}</p>}
    {!loading && !error && <div className="mt-7 grid gap-4 md:grid-cols-3">
      {plans.map((plan) => {
        const active = selected === plan.id;
        return <button key={plan.id} type="button" onClick={() => setSelected(plan.id)} aria-pressed={active} className={`min-w-0 rounded-xl border-2 p-4 text-left transition ${active ? "border-red-600 shadow-lg" : "border-gray-300 hover:border-gray-500"}`}>
          <div className={`rounded-lg p-4 text-white ${active ? "bg-linear-to-br from-red-600 to-red-950" : "bg-linear-to-br from-gray-700 to-gray-950"}`}>
            <div className="flex items-start justify-between gap-3"><div><h2 className="text-xl font-black">{plan.name}</h2><p className="mt-1 text-sm">{plan.quality}</p></div>{active && <span aria-hidden className="flex h-7 w-7 items-center justify-center rounded-full bg-white font-black text-red-600">✓</span>}</div>
          </div>
          <dl className="mt-4 divide-y divide-gray-200 text-sm">
            <div className="py-3"><dt className="text-gray-500">Monthly price</dt><dd className="mt-1 font-black">{formatCents(plan.price, plan.currency)}</dd></div>
            <div className="py-3"><dt className="text-gray-500">Video quality</dt><dd className="mt-1 font-bold">{plan.quality}</dd></div>
            <div className="py-3"><dt className="text-gray-500">Simultaneous streams</dt><dd className="mt-1 font-bold">{plan.max_streams}</dd></div>
          </dl>
        </button>;
      })}
    </div>}
    <div className="mt-8 flex justify-center"><button type="button" onClick={continueToPayment} disabled={!selected} className="min-h-14 w-full max-w-md rounded bg-red-600 px-8 text-xl font-bold text-white hover:bg-red-700 disabled:opacity-40">Next</button></div>
    <div className="mt-8 space-y-3 text-sm text-gray-600"><p>HD and Ultra HD availability depends on your internet service and device capabilities.</p><p>You may change or cancel your subscription at any time.</p></div>
  </CheckoutColumn></PaymentShell>;
}
