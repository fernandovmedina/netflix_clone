"use client";

import { CheckoutColumn, PaymentShell } from "@/components/payments/PaymentShell";
import Image from "next/image";
import { useRouter } from "next/navigation";

const methods = [
  { text: "Credit or Debit Card", type: "card", images: ["/payments/visa.png", "/payments/mastercard.png", "/payments/amex.png", "/payments/carnet.png"] },
  { text: "OXXO PAY", type: "oxxo", images: ["/payments/oxxo.png"] },
  { text: "Gift Code", type: "gift_code", images: ["/payments/gift_card.png"] },
];

export default function Payment() {
  const router = useRouter();
  return <PaymentShell><CheckoutColumn>
    <Image src="/lock.png" alt="Secure checkout" width={50} height={50} />
    <p className="mt-5 text-sm">STEP <strong>4</strong> OF <strong>4</strong></p>
    <h1 className="mt-1 text-3xl font-black">Choose how to pay.</h1>
    <p className="mt-2 text-sm">Your payment is encrypted. Cancel easily online.</p>
    <div className="mt-7 space-y-3">{methods.map((method) => <button key={method.type} type="button" onClick={() => router.push(`/signup/payment/${method.type}`)} className="flex min-h-16 w-full items-center justify-between gap-3 rounded border-2 border-gray-300 px-4 py-3 text-left hover:bg-gray-50 sm:px-5">
      <span className="min-w-0"><span className="block font-semibold">{method.text}</span><span className="mt-2 flex flex-wrap gap-1">{method.images.map((src) => <Image key={src} src={src} width={38} height={24} alt="" className="h-6 w-auto object-contain" />)}</span></span><span aria-hidden className="text-2xl">›</span>
    </button>)}</div>
    <p className="mt-5 text-xs text-gray-500">Gift Code applies a discount to Card or OXXO; it is not a separate payment rail.</p>
  </CheckoutColumn></PaymentShell>;
}
