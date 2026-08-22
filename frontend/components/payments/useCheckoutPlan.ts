"use client";

import { paymentApi, type Plan } from "@/utils/api/client";
import { rememberedCode, selectedPlanId } from "@/utils/payments";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export function useCheckoutPlan() {
  const router = useRouter();
  const [plan, setPlan] = useState<Plan | null>(null);
  const [code, setCode] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    setCode(rememberedCode());
    const id = selectedPlanId();
    if (!id) {
      router.replace("/signup/planform");
      setLoading(false);
      return;
    }
    paymentApi.plans().then((plans) => {
      const selected = plans.find((item) => item.id === id);
      if (!selected) {
        setError("The selected plan is no longer available.");
        return;
      }
      setPlan(selected);
    }).catch((reason: unknown) => setError(reason instanceof Error ? reason.message : "Unable to load the selected plan.")).finally(() => setLoading(false));
  }, [router]);

  return { plan, code, setCode, loading, error };
}
