"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function LinkRegistration() {
  const [savedEmail, setSavedEmail] = useState<string | null>(null);

  useEffect(() => {
    setSavedEmail(localStorage.getItem("signup_email"));
  }, []);

  const router: any = useRouter();

  const sendEmail = () => {
    router.push("/signup/linkSent")
  }

  return (
    <main className="bg-white text-black">
      <nav className="flex flex-row items-center justify-between py-7 border-b border-gray-600 w-full px-10">
        <Link href="/">
          <Image
            src="/netflix_logo.svg"
            alt="netflix_logo"
            width={160}
            height={30}
          />
        </Link>
        <Link href="/login" className="font-bold hover:underline text-xl">
          Sign In
        </Link>
      </nav>
      <section className="flex items-center justify-center mt-20 pb-44">
        <div className="w-[30%] flex mt-10 flex-col justify-center text-center">
          <Image
            className="mx-auto"
            src="/devices.png"
            alt="devices_image"
            width={250}
            height={20}
          />
          <p className="mt-10 text-sm">
            STEP <span className="font-bold">1</span> OF{" "}
            <span className="font-bold">4</span>
          </p>
          <h1 className="font-extrabold text-4xl">
            Finish setting up your account
          </h1>
          <p className="mt-5">
            We'll send a sign-up link to{" "}
            <span className="font-extrabold">{savedEmail}</span> so you can use
            Netflix without a password on any device at any time.
          </p>
          <button onClick={sendEmail} className="bg-red-600 hover:bg-red-400 text-white px-5 py-4 rounded mt-10 text-2xl font-extrabold">
            Send Link
          </button>
        </div>
      </section>
      <footer className="text-gray-800 py-15 px-40 bg-gray-700/40">
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
