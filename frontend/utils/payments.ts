import { ApiError, type Plan } from "@/utils/api/client";

export const SELECTED_PLAN_KEY = "signup_plan_id";
export const DISCOUNT_CODE_KEY = "signup_discount_code";

export function formatCents(cents: number, currency = "MXN"): string {
  const value = BigInt(Number.isInteger(cents) ? cents : 0);
  const zero = BigInt(0);
  const hundred = BigInt(100);
  const negative = value < zero;
  const absolute = negative ? -value : value;
  const whole = (absolute / hundred).toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  const fraction = (absolute % hundred).toString().padStart(2, "0");
  return `${currency} ${negative ? "-" : ""}${whole}.${fraction}`;
}

export function selectedPlanId(): number | null {
  if (typeof window === "undefined") return null;
  const value = Number(sessionStorage.getItem(SELECTED_PLAN_KEY));
  return Number.isInteger(value) && value > 0 ? value : null;
}

export function rememberPlan(plan: Plan): void {
  sessionStorage.setItem(SELECTED_PLAN_KEY, String(plan.id));
}

export function rememberedCode(): string {
  return typeof window === "undefined" ? "" : sessionStorage.getItem(DISCOUNT_CODE_KEY) ?? "";
}

export function rememberCode(code: string): void {
  const value = code.trim();
  if (value) sessionStorage.setItem(DISCOUNT_CODE_KEY, value);
  else sessionStorage.removeItem(DISCOUNT_CODE_KEY);
}

export function clearCheckoutState(): void {
  sessionStorage.removeItem(SELECTED_PLAN_KEY);
  sessionStorage.removeItem(DISCOUNT_CODE_KEY);
  localStorage.removeItem("plan_choosed");
  localStorage.removeItem("signup_giftcode");
}

export function paymentErrorMessage(reason: unknown): string {
  const message = reason instanceof Error ? reason.message.toLowerCase() : "";
  if (reason instanceof ApiError && reason.status === 409) return "That discount was redeemed by another payment attempt. Remove it or use a different code.";
  if (message.includes("redemption limit")) return "This code has already been redeemed or has reached its redemption limit.";
  if (message.includes("inactive") || message.includes("validity")) return "This code is invalid, inactive, or expired.";
  if (message.includes("not found") && message.includes("discount")) return "This gift code is invalid.";
  if (message.includes("card") || message.includes("declined")) return "The card was declined or its details are invalid. Check the fields or try another card.";
  if (message.includes("expired")) return "This payment or discount has expired. Start a new payment.";
  if (message.includes("plan")) return "The selected plan is no longer available. Choose another plan.";
  return reason instanceof Error ? reason.message : "Payment could not be completed.";
}
