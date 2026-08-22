"use client";
import Image from "next/image";
import Link from "next/link";
import { useState } from "react";

export default function LoginHelp() {
  const [emailPressed, setEmailPressed] = useState<boolean>(true);
  const [smsPressed, setSmsPressed] = useState<boolean>(false);
  const [forgot, setForgot] = useState<boolean>(false);
  const [email, setEmail] = useState<string>("");
  const [emailAlert, setEmailAlert] = useState<string>("");
  const [mobileNumberAlert, setMobileNumberAlert] = useState<string>("");
  const [mobileNumber, setMobileNumber] = useState<string>("");

  const toggle = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.name === "email") {
      setEmailPressed(true);
      setSmsPressed(false);
    } else {
      setEmailPressed(false);
      setSmsPressed(true);
    }
  };

  const verifyEmail = () => {
    if (email === "") {
      setEmailAlert("Please enter a valid email.");
    } else {
      setEmailAlert("");
    }
  }

  const verifyMobileNumber = () => {
    if (mobileNumber === "") {
      setMobileNumberAlert("Please enter a valid phone number.");
    } else {
      setMobileNumberAlert("");
    }
  }

  return (
    <main>
      <div className="min-h-dvh bg-[url(/home/hero_login.jpg)] bg-cover bg-center pb-10">
        <nav className="flex items-center justify-between border-b border-gray-400 px-5 py-5 text-white sm:px-10 lg:px-20">
          <Link href="/">
            <Image src="/netflix_logo.svg" width={150} height={41} alt="netflix_logo" />
          </Link>
          <Link href="/login" className="bg-red-600 rounded px-5 py-2 font-bold hover:bg-red-700">Sign In</Link>
        </nav>
        <div className={`${!forgot ? "text-black flex items-center justify-center" : "hidden"}`}>
          <div className="mx-4 mt-10 w-full max-w-lg bg-white px-5 py-10 sm:mt-20 sm:px-10">
            <h1 className="font-extrabold text-3xl">Update password, email or phone</h1>
            <p className="my-3 font-medium">How would you like to reset your password?</p>
            <div className="flex flex-row items-center pl-5">
              <label>
                <input name="email" type="checkbox" className="mr-2" checked={emailPressed} onChange={toggle} />
                Email
              </label>
            </div>
            <div className="flex flex-row items-center pl-5">
              <label>
                <input name="sms" type="checkbox" className="mr-2" checked={smsPressed} onChange={toggle} />
                Text Message (SMS)
              </label>
            </div>
            {
              emailPressed ?
              (
                <div className="mt-4">
                  <p>We will send you an email with instructions on how to reset your password.</p>
                  <input onChange={(e) => setEmail(e.target.value)} type="email" placeholder="Email" className="mt-3 border border-black rounded px-5 py-2 w-full text-left" />
                  <p className="text-red-600 text-xs my-2">{emailAlert}</p>
                  <button onClick={() => verifyEmail()} className="bg-red-600 hover:bg-red-700 w-full text-center py-2 rounded text-white font-bold mb-5">Email Me</button>
                </div>
              ) : (
                <div className="mt-4">
                  <p>We will text you a verification code to reset your password. Message and data rates may apply.</p>
                  <input onChange={(e) => setMobileNumber(e.target.value)} type="text" placeholder="Mobile number" className="mt-2 w-full border border-black rounded px-5 py-2 text-left" />
                  <p className="text-red-600 text-xs my-2">{mobileNumberAlert}</p>
                  <button onClick={() => verifyMobileNumber()} className="bg-red-600 hover:bg-red-700 w-full text-center text-white py-2 rounded font-bold mb-5">Text Me</button>
                </div>
              )
            }
            <p className="underline" onClick={() => setForgot(true)}>I don&apos;t remember my email or phone.</p>
          </div>
        </div>
        <div className={`${forgot ? "mt-10 flex items-center justify-center px-4 sm:mt-20" : "hidden"}`}>
          <div className="flex w-full max-w-lg flex-col bg-white px-5 py-10 text-black sm:px-10">
            <h1 className="font-extrabold text-3xl">Forgot email or mobile number</h1>
            <p className="my-5 text-sm">Please provide this information to help us find your account (all fields required):</p>
            <input className="min-h-12 rounded border border-gray-700 px-5 py-2 text-base" type="text" placeholder="First name on account" />
            <input className="my-4 min-h-12 rounded border border-gray-700 px-5 py-2 text-base" type="text" placeholder="Last name on account" />
            <input className="min-h-12 rounded border border-gray-700 px-5 py-2 text-base" type="text" placeholder="Credit or debit card number on file" />
            <div className="flex flex-row items-center mt-5">
              <button className="bg-red-600 rounded px-3 py-2 text-white font-bold mr-3 hover:bg-red-400">Find Account</button>
              <button onClick={() => setForgot(false)} className="bg-gray-600/50 rounded px-3 py-2 font-bold hover:bg-gray-600/30">Cancel</button>
            </div>
          </div>
        </div>
      </div>
      <footer className="bg-gray-700/40 px-5 py-10 text-gray-400 sm:px-10 lg:px-20">
        <a href="tel:8009539947">Questions? Call 800 953 9947 (Toll-Free)</a>
        <div className="mb-10 mt-5 grid w-full grid-cols-2 gap-4 text-sm sm:grid-cols-4">
          <div className="flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">FAQ</a>
            <a href="" className="hover:text-gray-200">Privacy</a>
          </div>
          <div className="flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">Help Center</a>
            <a href="" className="hover:text-gray-200">Cookie Preferences</a>
          </div>
          <div className="flex flex-col underline">
            <a href="" className="mb-2 hover:text-gray-200">Netflix Shop</a>
            <a href="" className="hover:text-gray-200">Corporate Information</a>
          </div>
          <div className="underline">
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
