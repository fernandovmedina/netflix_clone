"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function GiftCode() {
  const [planName, setPlanName] = useState<string>("");
  const [planPrice, setPlanPrice] = useState<number>(0);

  const [giftCode, setGiftCode] = useState<string>("");

  useEffect(() => {
    setGiftCode(localStorage.getItem("signup_giftcode") || "");
  }, []);

  useEffect(() => {
    localStorage.setItem("signup_giftcode", giftCode);
  }, [giftCode]);

  const router = useRouter();

  const changePlan = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();

    localStorage.setItem("signup_giftcode", giftCode);

    router.push("/signup/planform?change=true&paymentType=gift");
  };

  useEffect(() => {
    const plan = localStorage.getItem("plan_choosed");

    if (!plan) {
      alert("No plan selected, redirecting to plan selection.");
      router.push("/signup/planform");
      return;
    }

    switch (plan) {
      case "ads":
        setPlanName("Standard with ads");
        setPlanPrice(119);
        break;
      case "standard":
        setPlanName("Standard");
        setPlanPrice(249);
        break;
      case "premium":
        setPlanName("Premium");
        setPlanPrice(329);
        break;
      default:
        setPlanName("Premium");
        setPlanPrice(329);
        break;
    }
  }, [router]);

  return (
    <main className="bg-white text-black">
      <nav className="flex flex-row items-center justify-between py-7 border-b border-gray-600 w-full px-10">
        <Link href="/">
          <Image
            src="/netflix_logo.svg"
            alt="netflix_logo"
            width={160}
            height={43}
          />
        </Link>
        <Link href="/login" className="font-bold hover:underline text-xl">
          Sign Out
        </Link>
      </nav>
      <section className="flex items-center justify-center">
        <div className="w-[30%] mt-10 pb-24 flex flex-col">
          <Link
            className="flex flex-row items-center underline text-blue-700"
            href="/signup/payment"
          >
            <Image
              className="mr-2"
              src="/back_arrow.png"
              alt="back_arrow_icon"
              width={10}
              height={5}
            />
            Change payment method
          </Link>
          <p className="mt-5 text-sm">
            STEP <span className="font-bold">4</span> OF{" "}
            <span className="font-bold">4</span>
          </p>
          <h1 className="font-bold text-3xl">Enter your gift code</h1>
          <form>
            <input
              className="mt-2 border border-gray-400 rounded px-5 py-4 w-full"
              type="text"
              placeholder="Gift Card Pin or Code"
              onChange={(e) => setGiftCode(e.target.value)}
              value={giftCode}
            />
            <div className="bg-gray-400/20 my-5 rounded px-3 py-2 flex flex-row items-center justify-between">
              <div className="flex flex-col">
                <h1 className="font-bold">MXN {planPrice}/month</h1>
                <p className="text-sm text-gray-600">{planName}</p>
              </div>
              <button
                type="button"
                onClick={changePlan}
                className="text-blue-700 underline hover:text-blue-900"
              >
                Change
              </button>
            </div>
            <p className="text-gray-800/70 text-xs mb-4">
              By checking the checkbox below, you agree to our Terms of Use,
              Privacy Statement, and that you are over 18. Netflix will
              automatically continue your membership and charge the membership
              fee (currently MXN {planPrice}/month) to your payment method until
              you cancel. You may cancel at any time to avoid future charges.
            </p>
            <div className="flex flex-row items-center">
              <input type="checkbox" />
              <p className="ml-2 text-gray-800/80 text-sm">I agree</p>
            </div>
            <button className="bg-red-600 hover:bg-red-800 w-full rounded text-white font-bold py-4 my-5 text-2xl">
              Start Membership
            </button>
          </form>
        </div>
      </section>
    </main>
  );
}
