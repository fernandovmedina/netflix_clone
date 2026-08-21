"use client";

import AlertMessage from "@/components/AlertMessage";
import { useAuth } from "@/components/AuthProvider";
import { API_URL } from "@/utils/api/client";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function Login() {
  const [rememberMe, setRememberMe] = useState<boolean>(false);
  const [signInCode, setSignInCode] = useState<boolean>(false);
  const [email, setEmail] = useState<string>("");
  const [mobileNumber, setMobileNumber] = useState<string>("");
  const [password, setPassword] = useState<string>("");
  const [emailAlert, setEmailAlert] = useState<string>("");
  const [mobileNumberAlert, setMobileNumberAlert] = useState<string>("");
  const [passwordAlert, setPasswordAlert] = useState<string>("");
  const [isLoginError, setIsLoginError] = useState<boolean>(false);
  const [loginError, setLoginError] = useState("Email or password is incorrect.");
  const [submitting, setSubmitting] = useState(false);

  const router = useRouter();
  const { user, loading, login } = useAuth();

  useEffect(() => {
    if (!loading && user) {
      const next = new URLSearchParams(window.location.search).get("next");
      router.replace(next?.startsWith("/") ? next : "/home/browse");
    }
  }, [loading, router, user]);

  const handleSignIn = async (type: string) => {
    if (!verifySignIn(type)) return;
    setSubmitting(true);

    try {
      await login(email, password);
    } catch (error) {
      setLoginError(error instanceof Error ? error.message : "Unable to sign in.");
      setIsLoginError(true);
      return;
    } finally {
      setSubmitting(false);
    }

    const next = new URLSearchParams(window.location.search).get("next");
    router.replace(next?.startsWith("/") ? next : "/home/browse");
  }

  const toggleRememberMe = (e: React.ChangeEvent<HTMLInputElement>) => {
    setRememberMe(e.target.checked);
  };

  const verifySignIn = (type: string): boolean => {
    let valid = true;
    switch (type) {
      case "code":
        if (mobileNumber === "") {
          setMobileNumberAlert("Please enter a valid email or mobile number.");
          valid = false;
        } else {
          setMobileNumberAlert("");
        }
        break;
      case "password":
        if (email === "" && password === "") {
          setEmailAlert("Please enter a valid email or mobile number.");
          setPasswordAlert("Your password must contain between 4 and 60 characters.")
          valid = false;
        }
        if (email == "") {
          setEmailAlert("Please enter a valid email or mobile number.");
          valid = false;
        }
        if (password == "") {
          setPasswordAlert("Your password must contain between 4 and 60 characters.")
          valid = false;
        }
        if (email !== "") {
          setEmailAlert("");
        }
        if (password !== "") {
          setPasswordAlert("");
        }
    }
    return valid;
  }

  return (
    <main>
      <div className="bg-[url('/home/hero.jpg')] bg-cover bg-center px-40 h-screen">
        <nav className="py-5">
          <Link href="/">
            <Image
              src="/netflix_logo.svg"
              alt="netflix_logo"
              width={150}
              height={41}
            />
          </Link>
        </nav>
        <div className="flex items-center justify-center">
          <div className="text-white flex flex-col bg-black/90 mt-10 px-15 py-10 w-[40%]">
            <h1 className="font-extrabold text-3xl">Sign In</h1>
            <AlertMessage
              message={loginError}
              isOpened={isLoginError}
              onClose={() => setIsLoginError(false)}
            />
            {signInCode ? (
              <>
                <input onChange={(e) => setMobileNumber(e.target.value)} className="mt-5 border-2 border-gray-500 px-5 py-3 rounded placeholder:text-gray-300 bg-gray-900/50" type="email" placeholder="Email or mobile number" />
                <p className="text-xs text-red-600 mt-2">{mobileNumberAlert}</p>
              </>
            ) : (
              <>
                <input onChange={(e) => setEmail(e.target.value)} className="mt-5 border-2 border-gray-500 px-5 py-3 rounded placeholder:text-gray-300 bg-gray-900/50" type="email" placeholder="Email or mobile number" />
                <p className="text-xs text-red-600 mt-2">{emailAlert}</p>
                <input onChange={(e) => setPassword(e.target.value)} className={`${signInCode ? "hidden" : "mt-5 border-2 border-gray-500 px-5 py-3 rounded placeholder:text-gray-300 bg-gray-900/50"}`} type="password" placeholder="Password" />
                <p className="text-xs text-red-600 mt-2">{passwordAlert}</p>
              </>
            )}
            {signInCode ? (
              <button onClick={() => verifySignIn("code")} className="bg-red-600 my-4 py-2 font-bold rounded hover:bg-red-700 hover:cursor-pointer">Send Sign-In Code</button>
            ) : (
              <button disabled={submitting} onClick={() => handleSignIn("password")} className="bg-red-600 my-4 py-2 font-bold rounded hover:bg-red-700 hover:cursor-pointer disabled:cursor-wait disabled:opacity-60">{submitting ? "Signing In…" : "Sign In"}</button>
            )}
            <a href={`${API_URL}/api/v1/auth/google`} className="mb-4 rounded border border-gray-500 py-2 text-center font-semibold hover:bg-white/10">
              Continue with Google
            </a>
            <p className="text-center text-gray-400">OR</p>
            {signInCode ? (
              <button onClick={() => setSignInCode(false)} className="bg-gray-600/50 py-2 my-4 rounded hover:cursor-pointer hover:bg-gray-600/30">Use password</button>
            ) : (
              <button onClick={() => setSignInCode(true)} className="bg-gray-600/50 py-2 my-4 rounded hover:cursor-pointer hover:bg-gray-600/30">Use a Sign-In Code</button>
            )}
            <a href="/loginhelp" className="underline hover:text-gray-400 text-center">Forgot Password?</a>
            <div className="flex flex-row items-center my-4">
              <label className="flex flex-row items-center my-4 hover:cursor-pointer">
                <input
                  type="checkbox"
                  className="mr-3"
                  checked={rememberMe}
                  onChange={toggleRememberMe}
                />
                Remember me
              </label>
            </div>
            <div className="flex flex-row items-center mb-4">
              <p className="text-gray-400 mr-2">New to Netflix? {' '}</p>
              <Link href="/" className="font-bold">Sign up now</Link>
            </div>
            <p className="text-xs text-gray-400">This page is protected by Google reCAPTCHA to ensure you&apos;re not a bot.</p>
            <a className="text-blue-500 underline text-xs">Learn more.</a>
          </div>
        </div>
      </div>
      <footer className="text-gray-400 py-15 px-40 bg-gray-700/40">
        <a href="tel:8009539947">Questions? Call 800 953 9947 (Toll-Free)</a>
        <div className="mt-5 flex flex-row w-full mb-10 text-sm">
          <div className="w-1/4 flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">FAQ</a>
            <a href="" className="hover:text-gray-200">Privacy</a>
          </div>
          <div className="w-1/4 flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">Help Center</a>
            <a href="" className="hover:text-gray-200">Cookie Preferences</a>
          </div>
          <div className="w-1/4 flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">Netflix Shop</a>
            <a href="" className="hover:text-gray-200">Corporate Information</a>
          </div>
          <div className="w-1/4 underline">
            <a href="" className="hover:text-gray-200">Terms of Use</a>
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
