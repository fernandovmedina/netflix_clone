"use client";

import { useAuth } from "@/components/AuthProvider";
import { ArrowRightCircle, Edit2, Info, User } from "@deemlol/next-icons";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const navigation = [
  ["Home", "/home"],
  ["Series", "/home/series"],
  ["Movies", "/home/movies"],
  ["New & Popular", "/home/new_arrivals"],
  ["My List", "/home/my_list"],
] as const;

export const Navbar = () => {
  const [uuid, setUuid] = useState("");
  const [mobileOpen, setMobileOpen] = useState(false);
  const { user, logout } = useAuth();
  const router = useRouter();

  useEffect(() => {
    setUuid(new URLSearchParams(window.location.search).get("uuid") ?? "");
  }, []);

  const withProfile = (path: string) => (uuid ? `${path}?uuid=${encodeURIComponent(uuid)}` : path);
  const handleLogOut = async () => {
    await logout();
    router.replace("/login");
    router.refresh();
  };

  return (
    <nav className="relative flex w-full items-center gap-3 px-4 py-3 text-white sm:px-8 lg:px-14 lg:py-4">
      <Link href={withProfile("/home")} className="shrink-0">
        <Image src="/netflix_logo.svg" alt="Netflix" width={105} height={29} priority />
      </Link>
      <div className="hidden flex-1 items-center gap-5 px-3 text-sm md:flex">
        {navigation.map(([label, path]) => (
          <Link key={path} href={withProfile(path)} className="hover:text-gray-300">
            {label}
          </Link>
        ))}
      </div>
      <div className="ml-auto flex items-center gap-3">
        <Image src="/white_search.png" alt="Search" width={24} height={24} />
        <Image src="/white_notifications.png" alt="Notifications" width={24} height={24} className="hidden sm:block" />
        <div className="group relative hidden items-center gap-1 md:flex">
          <Image src="/gray_profile.png" alt="Profile" width={32} height={32} className="rounded" />
          <Image src="/white_dropdown.png" alt="Open profile menu" width={13} height={13} />
          <div className="absolute right-0 top-full h-3 w-56" />
          <div className="invisible absolute right-0 top-full z-50 mt-3 min-w-56 rounded bg-black/95 p-4 opacity-0 shadow-xl transition group-hover:visible group-hover:opacity-100">
            <div className="mb-3 flex items-center gap-3 border-b border-gray-700 pb-3">
              <Image src="/gray_profile.png" alt="" width={32} height={32} className="rounded" />
              <div><p className="text-sm font-bold">{user?.name || "Profile"}</p><p className="max-w-40 truncate text-xs text-gray-400">{user?.email}</p></div>
            </div>
            <Link href="/home/ManageProfiles" className="mb-3 flex items-center gap-3 text-sm hover:underline"><Edit2 size={20} /> Manage profiles</Link>
            <span className="mb-3 flex items-center gap-3 text-sm text-gray-400"><ArrowRightCircle size={20} /> Transfer profile</span>
            <span className="mb-3 flex items-center gap-3 text-sm text-gray-400"><User size={20} /> Account</span>
            <Link href="/loginhelp" className="mb-3 flex items-center gap-3 border-b border-gray-700 pb-3 text-sm hover:underline"><Info size={20} /> Help center</Link>
            <button type="button" onClick={handleLogOut} className="w-full pt-1 text-left text-sm hover:underline">Log out</button>
          </div>
        </div>
        <button type="button" onClick={() => setMobileOpen((value) => !value)} aria-expanded={mobileOpen} aria-controls="mobile-navigation" className="flex min-h-11 min-w-11 items-center justify-center rounded border border-white/40 bg-black/50 text-2xl md:hidden" aria-label="Toggle navigation menu">{mobileOpen ? "×" : "☰"}</button>
      </div>
      {mobileOpen && <div id="mobile-navigation" className="absolute inset-x-3 top-full z-50 mt-1 max-h-[calc(100dvh-5rem)] overflow-y-auto rounded-lg border border-zinc-700 bg-black/95 p-4 shadow-2xl md:hidden">
        <div className="grid gap-1">{navigation.map(([label, path]) => <Link key={path} href={withProfile(path)} onClick={() => setMobileOpen(false)} className="flex min-h-11 items-center rounded px-3 hover:bg-zinc-800">{label}</Link>)}</div>
        <div className="mt-3 border-t border-zinc-700 pt-3"><p className="px-3 text-sm font-bold">{user?.name || "Profile"}</p><p className="mb-2 truncate px-3 text-xs text-gray-400">{user?.email}</p><Link href="/home/ManageProfiles" onClick={() => setMobileOpen(false)} className="flex min-h-11 items-center rounded px-3 hover:bg-zinc-800">Manage profiles</Link><Link href="/loginhelp" onClick={() => setMobileOpen(false)} className="flex min-h-11 items-center rounded px-3 hover:bg-zinc-800">Help center</Link><button type="button" onClick={handleLogOut} className="min-h-11 w-full rounded px-3 text-left hover:bg-zinc-800">Log out</button></div>
      </div>}
    </nav>
  );
};
