"use client";

import { useAuth } from "@/components/AuthProvider";
import { artworkUrl, userApi, type Profile } from "@/utils/api/client";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useState } from "react";

export default function Browse() {
  const { user } = useAuth();
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [isAddProfileModalOpen, setIsAddProfileModalOpen] = useState(false);

  useEffect(() => {
    userApi
      .profiles()
      .then(setProfiles)
      .catch((reason: unknown) => setError(reason instanceof Error ? reason.message : "Unable to load profiles."))
      .finally(() => setLoading(false));
  }, []);

  const visibleProfiles = profiles.length > 0 ? profiles : user ? [{ id: user.id, name: user.name || "Main profile", avatar: "/gray_profile.png", is_kids: false }] : [];

  return (
    <main className="flex min-h-screen items-center justify-center bg-black px-5 text-white">
      <div className="text-center">
        <h1 className="text-3xl sm:text-5xl">Who&apos;s watching now?</h1>
        {loading && <p className="mt-8 animate-pulse text-gray-400">Loading profiles…</p>}
        {error && <p className="mt-5 text-sm text-amber-300">{error} Showing your main profile.</p>}
        <div className="mt-8 flex flex-wrap items-start justify-center gap-5">
          {visibleProfiles.map((profile) => (
            <Link key={profile.id} href={`/home?uuid=${encodeURIComponent(profile.id)}`} className="group flex w-28 flex-col items-center">
              <Image src={profile.avatar ? artworkUrl(profile.avatar) : "/gray_profile.png"} alt={`${profile.name} profile`} width={112} height={112} className="aspect-square rounded object-cover ring-2 ring-transparent group-hover:ring-white" unoptimized={Boolean(profile.avatar?.startsWith("/"))} />
              <span className="mt-2 text-gray-400 group-hover:text-white">{profile.name}</span>
            </Link>
          ))}
          <button type="button" onClick={() => setIsAddProfileModalOpen(true)} className="group flex w-28 flex-col items-center">
            <span className="flex aspect-square w-28 items-center justify-center rounded border border-dashed border-gray-400 group-hover:border-white"><Image src="/add_white.png" alt="" width={32} height={32} /></span>
            <span className="mt-2 text-gray-400 group-hover:text-white">Add profile</span>
          </button>
        </div>
        <Link href="/home/ManageProfiles" className="mt-10 inline-block border-2 border-gray-500 px-5 py-2 text-gray-400 hover:border-white hover:text-white">Manage Profiles</Link>
      </div>
      {isAddProfileModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 sm:p-4">
          <div className="relative h-dvh w-full overflow-y-auto bg-[#181818] p-6 text-left sm:h-auto sm:max-h-[90dvh] sm:max-w-xl sm:rounded-lg sm:p-10">
            <button type="button" onClick={() => setIsAddProfileModalOpen(false)} className="absolute right-4 top-4 flex h-11 w-11 items-center justify-center text-3xl" aria-label="Close add-profile sheet">×</button>
            <h2 className="text-2xl font-bold">Add a profile</h2>
            <p className="mt-2 text-sm text-gray-300">Profile creation will be available when account management is enabled.</p>
            <div className="mt-6 flex items-center gap-4"><Image src="/gray_profile.png" alt="" width={70} height={70} /><input className="min-w-0 flex-1 rounded border border-gray-500 bg-transparent px-4 py-3 text-base" placeholder="Name" /></div>
            <button type="button" onClick={() => setIsAddProfileModalOpen(false)} className="mt-7 w-full rounded bg-white py-3 font-bold text-black">Done</button>
          </div>
        </div>
      )}
    </main>
  );
}
