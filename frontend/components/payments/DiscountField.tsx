"use client";

import { paymentApi, type DiscountPreview, type Plan } from "@/utils/api/client";
import { formatCents, paymentErrorMessage, rememberCode } from "@/utils/payments";
import { useState } from "react";

type DiscountFieldProps = {
  plan: Plan;
  code: string;
  onCodeChange: (code: string) => void;
  onPreview: (preview: DiscountPreview | null) => void;
};

export function DiscountField({ plan, code, onCodeChange, onPreview }: DiscountFieldProps) {
  const [preview, setPreview] = useState<DiscountPreview | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const updateCode = (value: string) => {
    onCodeChange(value);
    setPreview(null);
    onPreview(null);
    setError("");
  };

  const validate = async () => {
    const normalized = code.trim();
    if (!normalized) {
      rememberCode("");
      setPreview(null);
      onPreview(null);
      setError("Enter a gift code to preview its discount.");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const result = await paymentApi.previewDiscount(plan.id, normalized);
      setPreview(result);
      onPreview(result);
      if (result.valid) rememberCode(normalized);
      else {
        rememberCode("");
        setError(paymentErrorMessage(new Error(result.reason ?? "This code is invalid or unavailable.")));
      }
    } catch (reason) {
      setError(paymentErrorMessage(reason));
      setPreview(null);
      onPreview(null);
    } finally {
      setLoading(false);
    }
  };

  return <div className="mt-5 rounded-lg border border-gray-300 p-4">
    <label htmlFor="discount-code" className="text-sm font-bold">Gift code <span className="font-normal text-gray-500">(optional discount)</span></label>
    <div className="mt-2 flex flex-col gap-2 sm:flex-row">
      <input id="discount-code" value={code} onChange={(event) => updateCode(event.target.value.toUpperCase())} autoComplete="off" className="min-h-12 min-w-0 flex-1 rounded border border-gray-400 px-3 text-base uppercase" placeholder="Enter code" />
      <button type="button" onClick={validate} disabled={loading} className="min-h-12 rounded border border-gray-800 px-5 font-bold disabled:opacity-50">{loading ? "Checking…" : "Apply"}</button>
    </div>
    {error && <p role="alert" className="mt-2 text-sm text-red-700">{error}</p>}
    {preview?.valid && <div className="mt-3 grid grid-cols-2 gap-1 rounded bg-emerald-50 p-3 text-sm">
      <span>Subtotal</span><span className="text-right">{formatCents(preview.subtotal, preview.currency)}</span>
      <span>Discount</span><span className="text-right text-emerald-700">−{formatCents(preview.discount_amount, preview.currency)}</span>
      <strong>Preview total</strong><strong className="text-right">{formatCents(preview.total, preview.currency)}</strong>
    </div>}
    <p className="mt-2 text-xs text-gray-500">This preview is advisory. The server confirms the final amount when payment is created.</p>
  </div>;
}
