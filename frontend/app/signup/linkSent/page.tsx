"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function LinkSent() {
  const [savedEmail, setSavedEmail] = useState<string | null>(null);

  useEffect(() => {
    setSavedEmail(localStorage.getItem("signup_email"));
  }, []);

  const router = useRouter();

  const createPassword = () => {
    router.push("/signup/regform")
  };

  return (
    <main className="bg-white text-black">
      <nav className="flex w-full items-center justify-between border-b border-gray-300 px-4 py-5 sm:px-10 sm:py-7">
        <Link href="/">
          <Image
            src="/netflix_logo.svg"
            alt="netflix_logo"
            width={160}
            height={43}
          />
        </Link>
        <Link href="/login" className="font-bold hover:underline text-xl">
          Sign In
        </Link>
      </nav>
      <section className="flex items-center justify-center">
        <div className="mt-12 flex w-full max-w-lg flex-col justify-center px-5 pb-20 text-center sm:mt-20">
          <Image
            className="mx-auto"
            src="/packet.png"
            alt="packet_image"
            width={50}
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
            We&apos;ll send a sign-up link to{" "}
            <span className="font-extrabold">{savedEmail}</span> so you can use
            Netflix without a password on any device at any time.
          </p>
          <button
            onClick={() => {}}
            className="bg-red-600 hover:bg-red-400 text-white px-5 py-4 rounded mt-10 text-2xl font-extrabold"
          >
            Resend Link
          </button>
					<button onClick={createPassword} className="bg-gray-500/30 text-2xl hover:bg-gray-500/50 mt-5 text-black font-extrabold px-5 py-4 rounded">
						Create Password Instead
					</button>
        </div>
      </section>
      <footer className="bg-gray-100 px-5 py-10 text-gray-800 sm:px-10 lg:px-20">
        <a href="tel:8009539947">Questions? Call 800 953 9947 (Toll-Free)</a>
        <div className="mb-10 mt-5 grid w-full grid-cols-2 gap-4 text-sm sm:grid-cols-4">
          <div className="flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">
              FAQ
            </a>
            <a href="" className="hover:text-gray-200">
              Privacy
            </a>
          </div>
          <div className="flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">
              Help Center
            </a>
            <a href="" className="hover:text-gray-200">
              Cookie Preferences
            </a>
          </div>
          <div className="flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">
              Netflix Shop
            </a>
            <a href="" className="hover:text-gray-200">
              Corporate Information
            </a>
          </div>
          <div className="underline">
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
