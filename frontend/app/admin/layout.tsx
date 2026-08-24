"use client";

import { useAuth } from "@/components/AuthProvider";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && user?.role !== "admin") router.replace(user ? "/home" : "/login?next=/admin");
  }, [loading, router, user]);

  if (loading) return <main className="flex min-h-dvh items-center justify-center bg-black text-white"><p className="animate-pulse">Checking administrator access…</p></main>;
  if (user?.role !== "admin") return <main className="flex min-h-dvh items-center justify-center bg-black px-6 text-center text-white"><div><h1 className="text-3xl font-black">Administrator access required</h1><p className="mt-3 text-zinc-400">Redirecting you to a safe page…</p></div></main>;

  return <div className="min-h-dvh bg-black text-white">
    <header className="sticky top-0 z-40 border-b border-zinc-800 bg-black/95 backdrop-blur">
      <nav className="mx-auto flex max-w-7xl flex-wrap items-center gap-3 px-4 py-4 sm:flex-nowrap sm:gap-5 sm:px-7">
        <Link href="/admin"><Image src="/netflix_logo.svg" alt="Netflix" width={108} height={30} priority /></Link>
        <span className="rounded bg-red-600 px-2 py-1 text-xs font-black uppercase tracking-wider">Admin</span>
        <Link href="/admin" className="ml-auto text-sm text-zinc-300 hover:text-white">Titles</Link>
        <Link href="/home" className="text-sm text-zinc-300 hover:text-white">← Back to browse</Link>
        <button type="button" onClick={async () => { await logout(); router.replace("/login"); }} className="text-sm text-zinc-300 hover:text-white">Log out</button>
      </nav>
    </header>
    <main className="mx-auto max-w-7xl overflow-x-hidden px-4 py-8 sm:px-7 sm:py-12">{children}</main>
  </div>;
}
