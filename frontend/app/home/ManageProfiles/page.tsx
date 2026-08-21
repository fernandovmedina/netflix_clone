"use client";

import { artworkUrl, userApi, type Profile } from "@/utils/api/client";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useState } from "react";

export default function ManageProfiles() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    userApi
      .profiles()
      .then(setProfiles)
      .catch((reason: unknown) => setError(reason instanceof Error ? reason.message : "Unable to load profiles."))
      .finally(() => setLoading(false));
  }, []);

  return (
    <main className="flex min-h-screen items-center justify-center bg-black px-5 text-white">
      <div className="text-center">
        <h1 className="text-4xl sm:text-5xl">Manage profiles</h1>
        {loading && <p className="mt-8 animate-pulse text-gray-400">Loading profiles…</p>}
        {error && <p className="mt-5 text-red-300">{error}</p>}
        {!loading && !error && profiles.length === 0 && <p className="mt-5 text-gray-400">No profiles have been created yet.</p>}
        <div className="my-8 flex flex-wrap justify-center gap-5">
          {profiles.map((profile) => (
            <Link key={profile.id} href={`/home/settings/${profile.id}`} className="group flex w-28 flex-col items-center">
              <span className="relative"><Image src={profile.avatar ? artworkUrl(profile.avatar) : "/gray_profile.png"} alt={profile.name} width={112} height={112} className="aspect-square rounded object-cover opacity-75 group-hover:opacity-100" unoptimized={Boolean(profile.avatar?.startsWith("/"))} /><Image src="/edit_white.png" alt="Edit" width={30} height={30} className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2" /></span>
              <span className="mt-2 text-gray-400 group-hover:text-white">{profile.name}</span>
            </Link>
          ))}
        </div>
        <Link href="/home/browse" className="inline-block bg-white px-6 py-2 font-bold text-black hover:bg-red-600 hover:text-white">Done</Link>
      </div>
    </main>
  );
}
