"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

export default function Card() {
  const cardInputRef = useRef<HTMLInputElement>(null);
  const cvvInputRef = useRef<HTMLInputElement>(null);

  const [planName, setPlanName] = useState<string>("");
  const [planPrice, setPlanPrice] = useState<number>(0);

  const [isAgreed, setIsAgreed] = useState<boolean>(false);

  const [cardNumber, setCardNumber] = useState<string>("");
  const [cardExpiration, setCardExpiration] = useState<string>("");
  const [cvv, setCvv] = useState<string>("");
  const [cardName, setCardName] = useState<string>("");

  useEffect(() => {
    setCardNumber(localStorage.getItem("signup_cardnumber") || "");
    setCardExpiration(localStorage.getItem("signup_cardexpiration") || "");
    setCardName(localStorage.getItem("signup_cardname") || "");
  }, []);

  const router = useRouter();

  const startMembership = (e: React.MouseEvent<HTMLButtonElement>) => {
    if (!isAgreed) {
      e.preventDefault();
      alert("You must agree to the terms to start membership.");
      return;
    } else {
      // TODO: Process payment and start membership
      // TODO: Save user payment info to database
      localStorage.clear();

      router.push("/");
    }
  };

  const changePlan = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();

    localStorage.setItem("signup_cardnumber", cardNumber);
    localStorage.setItem("signup_cardexpiration", cardExpiration);
    localStorage.setItem("signup_cardname", cardName);

    router.push("/signup/planform?change=true&paymentType=card");
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
  }, []);

  const toggleInput = (name: string) => {
    if (name === "card") {
      cardInputRef.current?.focus();
    } else {
      cvvInputRef.current?.focus();
    }
  };

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
          <h1 className="font-bold text-3xl">
            Set up your credit or debit card.
          </h1>
          <div className="flex flex-row items-center mt-6">
            <Image
              src="/payments/visa.png"
              alt="payment_image"
              width={45}
              height={5}
            />
            <Image
              className="mx-1"
              src="/payments/mastercard.png"
              alt="payment_image"
              width={45}
              height={5}
            />
            <Image
              src="/payments/amex.png"
              alt="payment_image"
              width={45}
              height={5}
            />
            <Image
              className="ml-1"
              src="/payments/carnet.png"
              alt="payment_image"
              width={45}
              height={5}
            />
          </div>
          <form>
            <div
              onClick={() => toggleInput("card")}
              className="flex flex-row w-full items-center border my-3 border-gray-400 rounded px-5 py-2 justify-between"
            >
              <input
                ref={cardInputRef}
                id="card_input"
                type="text"
                value={cardNumber}
                placeholder="Card number"
                className="w-[85%] outline-none"
                onChange={(e) => {
                  let value = e.target.value;

                  value = value.replace(/\D/g, "");

                  if (value.length > 16) value = value.slice(0, 16);

                  value = value.replace(/(.{4})/g, "$1 ").trim();

                  setCardNumber(value);
                }}
              />
              <Image
                src="/debit_card.png"
                width={30}
                height={5}
                alt="debit_card_icon"
                className="w-[10%]"
              />
            </div>
            <div className="mt-4 flex flex-row items-center gap-4">
              <input
                className="border border-gray-400 rounded px-5 py-4 w-1/2"
                type="text"
                placeholder="Expiration date"
                value={cardExpiration}
                onChange={(e) => {
                  let value = e.target.value;

                  value = value.replace(/\D/g, "");

                  if (value.length > 4) value = value.slice(0, 4);

                  if (value.length > 2) {
                    value = value.slice(0, 2) + "/" + value.slice(2);
                  }

                  setCardExpiration(value);
                }}
              />
              <div
                onClick={() => toggleInput("cvv")}
                className="flex flex-row items-center border border-gray-400 rounded px-5 py-4 w-1/2 justify-between"
              >
                <input
                  ref={cvvInputRef}
                  id="cvv_input"
                  type="number"
                  onChange={(e) => setCvv(e.target.value)}
                  value={cvv}
                  placeholder="CVV"
                  className="w-full outline-none"
                />
                <Image
                  src="/info.png"
                  width={25}
                  height={25}
                  alt="info_icon"
                  className="ml-3"
                />
              </div>
            </div>
            <input
              className="mt-4 border border-gray-400 rounded px-5 py-4 w-full"
              type="text"
              onChange={(e) => setCardName(e.target.value)}
              value={cardName}
              placeholder="Name on card"
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
            <p className="text-gray-800/70 text-sm mb-6">
              Your payments will be processed internationally. Additional bank
              fees may apply.
            </p>
            <p className="text-gray-800/70 text-xs mb-4">
              By checking the checkbox below, you agree to our Terms of Use,
              Privacy Statement, and that you are over 18. Netflix will
              automatically continue your membership and charge the membership
              fee (currently MXN 119/month) to your payment method until you
              cancel. You may cancel at any time to avoid future charges.
            </p>
            <div className="flex flex-row items-center">
              <input
                type="checkbox"
                checked={isAgreed}
                onChange={(e) => setIsAgreed(e.target.checked)}
              />
              <p className="ml-2 text-gray-800/80 text-sm">I agree</p>
            </div>
            <button
              type="submit"
              onClick={startMembership}
              className="bg-red-600 hover:bg-red-800 w-full rounded text-white font-bold py-4 my-5 text-2xl"
            >
              Start Membership
            </button>
          </form>
        </div>
      </section>
    </main>
  );
}
