"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function Planform() {
  const [planChoosed, setPlanChoosed] = useState("premium");
  const [paymentType, setPaymentType] = useState<string | null>(null);
  const [change, setChange] = useState<string | null>(null);

  const router = useRouter();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);

    setPaymentType(params.get("paymentType"));
    setChange(params.get("change"));
  }, []);

  const goPay = () => {
    localStorage.setItem("plan_choosed", planChoosed);

    if (change) {
      switch (paymentType) {
        case "card":
          router.push(`/signup/payment/card`);
          break;
        case "gift":
          router.push(`/signup/payment/gift_code`);
          break;
        case "oxxo":
          router.push(`/signup/payment/oxxo`);
          break;
        default:
          router.push("/signup/payment");
      }
      return;
    }
    
    router.push("/signup/payment");
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
      <section className="px-40">
        <p className="mt-10 text-sm">
          STEP <span className="font-bold">3</span> OF{" "}
          <span className="font-bold">4</span>
        </p>
        <h1 className="text-4xl font-bold">
          Choose the plan that&apos;s right for you
        </h1>
        <div className="mt-5 mb-20 flex flex-row w-full">
          <div
            onClick={() => setPlanChoosed("ads")}
            className="w-1/3 flex flex-col p-3 border border-gray-700 rounded-xl hover:cursor-pointer"
          >
            <div className="bg-linear-to-r rounded px-2 pt-3 pb-2 from-sky-800 via-cyan-800 to-sky-950">
              <h1 className="text-white font-bold">Standard with ads</h1>
              <p className="text-white pb-3 text-sm">1080p</p>
              {planChoosed == "ads" && (
                <div className="flex justify-end">
                  <Image
                    src="/verify_white.png"
                    alt="verify_icon"
                    width={20}
                    height={10}
                  />
                </div>
              )}
            </div>
            <div className="mt-7 border-b border-gray-700">
              <h2 className="text-gray-700 font-medium text-sm">
                Monthly price
              </h2>
              <p className="text-base font-extrabold pb-2">MXN 119</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Video and sound quality
              </h2>
              <p className="text-base font-extrabold pb-2">Good</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">Resolution</h2>
              <p className="text-base font-extrabold pb-2">1080p (Full HD)</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Supported devices
              </h2>
              <p className="text-base font-extrabold pb-2">
                TV, computer, mobile phone, tablet
              </p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Devices your household can watch at the same time
              </h2>
              <p className="text-base font-extrabold pb-2">2</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Download devices
              </h2>
              <p className="text-base font-extrabold pb-2">2</p>
            </div>
            <div className="mt-3">
              <h2 className="text-gray-700 font-medium text-sm">Ads</h2>
              <p className="text-base font-extrabold pb-2">
                Less than you might think
              </p>
            </div>
          </div>
          <div
            onClick={() => setPlanChoosed("standard")}
            className="w-1/3 hover:cursor-pointer flex flex-col ml-2 p-3 border border-gray-700 rounded-xl"
          >
            <div className="bg-linear-to-r rounded px-2 pt-3 pb-2 from-sky-800 via-cyan-800 to-sky-950">
              <h1 className="text-white font-bold">Standard</h1>
              <p className="text-white pb-3 text-sm">1080p</p>
              {planChoosed == "standard" && (
                <div className="flex justify-end">
                  <Image
                    src="/verify_white.png"
                    alt="verify_icon"
                    width={20}
                    height={10}
                  />
                </div>
              )}
            </div>
            <div className="mt-7 border-b border-gray-700">
              <h2 className="text-gray-700 font-medium text-sm">
                Monthly price
              </h2>
              <p className="text-base font-extrabold pb-2">MXN 249</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Video and sound quality
              </h2>
              <p className="text-base font-extrabold pb-2">Good</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">Resolution</h2>
              <p className="text-base font-extrabold pb-2">1080p (Full HD)</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Supported devices
              </h2>
              <p className="text-base font-extrabold pb-2">
                TV, computer, mobile phone, tablet
              </p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Devices your household can watch at the same time
              </h2>
              <p className="text-base font-extrabold pb-2">2</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Download devices
              </h2>
              <p className="text-base font-extrabold pb-2">2</p>
            </div>
            <div className="mt-3">
              <h2 className="text-gray-700 font-medium text-sm">Ads</h2>
              <p className="text-base font-extrabold pb-2">No ads</p>
            </div>
          </div>
          <div
            onClick={() => setPlanChoosed("premium")}
            className="w-1/3 hover:cursor-pointer flex flex-col ml-2 p-3 border border-gray-700 rounded-xl"
          >
            <div className="bg-linear-to-r rounded px-2 pt-3 pb-2 from-fuchsia-800 to-pink-950">
              <h1 className="text-white font-bold">Premium</h1>
              <p className="text-white pb-3 text-sm">4K + HDR</p>
              {planChoosed == "premium" && (
                <div className="flex justify-end">
                  <Image
                    src="/verify_white.png"
                    alt="verify_icon"
                    width={20}
                    height={10}
                  />
                </div>
              )}
            </div>
            <div className="mt-7 border-b border-gray-700">
              <h2 className="text-gray-700 font-medium text-sm">
                Monthly price
              </h2>
              <p className="text-base font-extrabold pb-2">MXN 329</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Video and sound quality
              </h2>
              <p className="text-base font-extrabold pb-2">Best</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">Resolution</h2>
              <p className="text-base font-extrabold pb-2">
                4K (Ultra HD) + HDR
              </p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Supported devices
              </h2>
              <p className="text-base font-extrabold pb-2">
                TV, computer, mobile phone, tablet
              </p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Devices your household can watch at the same time
              </h2>
              <p className="text-base font-extrabold pb-2">4</p>
            </div>
            <div className="border-b border-gray-700 mt-3">
              <h2 className="text-gray-700 font-medium text-sm">
                Download devices
              </h2>
              <p className="text-base font-extrabold pb-2">2</p>
            </div>
            <div className="mt-3">
              <h2 className="text-gray-700 font-medium text-sm">Ads</h2>
              <p className="text-base font-extrabold pb-2">No ads</p>
            </div>
          </div>
        </div>
        <div className="pb-10 text-sm">
          <p>
            Learn more about an ad-supported plan. If you select an ad-supported
            plan, you will be required to provide your date of birth for ads
            personalization and other purposes consistent with the Netflix
            Privacy Statement.
          </p>
          <p className="my-3">
            Full HD (1080p), Ultra HD (4K) and HDR avalability subjet to your
            internet service and device capabilities. Not all content is
            available in all resolutions. See Terms of Use for more details.
          </p>
          <p>
            Only people who live with you may use your account. Add 1 extra
            member with Standard or up to 2 with Premium. Learn more. Watch on 4
            different devices at the same time with remium and 2 with Standard
            or Standard with ads.
          </p>
          <p className="mt-3 mb-8">
            Live events are included with any Netflix plan and contain ads.
          </p>
          <div className="flex justify-center">
            <button
              onClick={() => goPay()}
              className="bg-red-600 text-white hover:bg-red-400 w-[30%] font-bold text-xl py-5 rounded"
            >
              NEXT
            </button>
          </div>
        </div>
      </section>
      <footer className="text-gray-800 py-15 px-40 bg-gray-400/40">
        <a href="tel:8009539947">Questions? Call 800 953 9947 (Toll-Free)</a>
        <div className="mt-5 flex flex-row w-full mb-10 text-sm">
          <div className="w-1/4 flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">
              FAQ
            </a>
            <a href="" className="hover:text-gray-200">
              Privacy
            </a>
          </div>
          <div className="w-1/4 flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">
              Help Center
            </a>
            <a href="" className="hover:text-gray-200">
              Cookie Preferences
            </a>
          </div>
          <div className="w-1/4 flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">
              Netflix Shop
            </a>
            <a href="" className="hover:text-gray-200">
              Corporate Information
            </a>
          </div>
          <div className="w-1/4 underline">
            <a href="" className="hover:text-gray-200">
              Terms of Use
            </a>
          </div>
        </div>
        <div className="flex flex-row border-2 border-gray-400 bg-black/60 px-2 rounded-lg w-36">
          <Image
            src="/language.png"
            alt="language_icon"
            width={35}
            height={0}
          />
          <select className="text-white pl-2 bg-transparent font-semibold">
            <option>English</option>
            <option>Español</option>
          </select>
        </div>
      </footer>
    </main>
  );
}
