import { formatCents } from "@/utils/payments";

export function PaymentBreakdown({ subtotal, discount, total, currency, authoritative = false }: { subtotal: number; discount: number; total: number; currency: string; authoritative?: boolean }) {
  return <div className={`mt-5 rounded-lg p-4 ${authoritative ? "border border-emerald-300 bg-emerald-50" : "bg-gray-100"}`}>
    <div className="grid grid-cols-2 gap-y-2 text-sm">
      <span>Subtotal</span><span className="text-right">{formatCents(subtotal, currency)}</span>
      <span>Discount applied</span><span className="text-right">−{formatCents(discount, currency)}</span>
      <strong className="border-t border-gray-300 pt-2">Total</strong><strong className="border-t border-gray-300 pt-2 text-right">{formatCents(total, currency)}</strong>
    </div>
    {authoritative && <p className="mt-3 text-xs font-semibold text-emerald-800">Authoritative total confirmed by the payment server.</p>}
  </div>;
}
