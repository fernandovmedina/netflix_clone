"use client";

import AlertMessage from "@/components/AlertMessage";
import { signup } from "@/utils/api/auth";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function Regform() {
  const [password, setPassword] = useState<string>("");
  const [email, setEmail] = useState<string>("");
  const [signupAlert, setSignupAlert] = useState<string>("");
  const [isSignupError, setIsSignupError] = useState<boolean>(false);

  useEffect(() => {
    setEmail(localStorage.getItem("signup_email") ?? "");
  }, []);

  const router: any = useRouter();

  const handleSignup = async () => {
    if (email === "" || password === "") {
      setSignupAlert("Please enter your email and a password.");
      setIsSignupError(true);
      return;
    }

    try {
      // Creates the account through the auth service (nginx load balancer);
      // Supabase then sends the confirmation email the next step refers to.
      await signup("", email, password);
    } catch (error) {
      setSignupAlert(error instanceof Error ? error.message : "Something went wrong. Please try again.");
      setIsSignupError(true);
      return;
    }

    localStorage.setItem("signup_email", email);
    router.push("/signup/verifyEmail");
  }

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
          Sign In
        </Link>
      </nav>
      <section className="flex items-center justify-center">
        <div className="w-[30%] mt-20 pb-44 flex flex-col justify-center">
          <p className="mt-10 text-sm">
            STEP <span className="font-bold">1</span> OF{" "}
            <span className="font-bold">4</span>
          </p>
          <h1 className="font-bold text-4xl">
            Create a password to start your membership
          </h1>
          <p className="mt-5">
            Just a few more steps and you're done! We hate paperwork, too.
          </p>
          <AlertMessage
            message={signupAlert}
            isOpened={isSignupError}
            onClose={() => setIsSignupError(false)}
          />
          <input onChange={(e) => setEmail(e.target.value)} value={email} className="border border-gray-400 px-4 py-4 mt-4 rounded" type="email" placeholder="email@email.com" />
          <input onChange={(e) => setPassword(e.target.value)} className="border border-gray-400 px-4 py-4 rounded mt-2" type="password" placeholder="Add a password" />
          <button
            onClick={handleSignup}
            className="bg-red-600 hover:bg-red-400 text-white px-5 py-4 rounded mt-5 text-2xl font-extrabold"
          >
            Next
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
