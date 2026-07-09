"use client";

import Image from "next/image";
import { useRouter } from "next/navigation";
import { useState } from "react";

export default function ProfileSettings() {
  const router = useRouter();

  const [isMainProfile, setIsMainProfile] = useState<boolean>(true);

  return (
    <main className="bg-white text-black">
      <nav className="py-5 border-b border-gray-300 mb-10 px-48">
        <Image
          src="/netflix_logo.svg"
          alt="Netflix Logo"
          width={120}
          height={32}
        />
      </nav>
      <section className="flex flex-col mx-auto justify-center w-[800px]">
        <div className="flex flex-row items-center">
          <Image
            onClick={() => router.back()}
            src="/back_arrow.png"
            alt="Back Arrow"
            width={30}
            height={20}
            className="hover:cursor-pointer"
          />
          <h1 className="text-3xl font-bold ml-2">
            Manage profile and preferences
          </h1>
        </div>
        <div className="flex flex-col mt-10">
          <div className="border-2 p-5 border-gray-500/30 rounded-lg">
            <div className="flex items-center justify-between pb-5">
              <div className="flex items-center gap-4">
                <Image
                  src="https://occ-0-7553-114.1.nflxso.net/dnm/api/v6/vN7bi_My87NPKvsBoib006Llxzg/AAAABTIGL_gJccfoM0OVmMb1xD1VgUNeiW5QN0r0ac7yKqHBmrnO92ctrpMKE6v2lkMIRp_vYmREKLtTJ2lodWUtiG5wdnPQGjjXvg.png?r=18a"
                  alt="profile_image"
                  width={40}
                  height={40}
                  className="rounded object-cover"
                />
                <div className="flex flex-col">
                  <h1 className="font-semibold text-lg">Fernando</h1>
                  <p className="text-sm text-gray-500">
                    Editar información personal y de contacto
                  </p>
                </div>
              </div>
              <Image
                src="/front_arrow.png"
                alt="front_arrow"
                width={20}
                height={20}
                className="opacity-70"
              />
            </div>
            <hr />
            <div className="flex items-center justify-between pt-5">
              <div className="flex items-center gap-4">
                <Image
                  src="/gray_block.png"
                  alt="profile_image"
                  width={25}
                  height={48}
                  className="rounded object-cover"
                />
                <div className="flex flex-col">
                  <h1 className="font-semibold text-lg">Bloqueo de perfil</h1>
                  <p className="text-sm text-gray-500">
                    Solicita un PIN para acceder este perfil
                  </p>
                </div>
              </div>
              <Image
                src="/front_arrow.png"
                alt="front_arrow"
                width={20}
                height={20}
                className="opacity-70"
              />
            </div>
          </div>
          <h1 className="mt-10 text-xl mb-4">Preferences</h1>
          <div className="border-2 px-5 border-gray-500/30 rounded-lg">
            <div className="flex items-center justify-between py-2 my-3 px-5 hover:bg-gray-500/10">
              <div className="flex items-center gap-4">
                <Image
                  src="/gray_language.png"
                  alt="profile_image"
                  width={27}
                  height={27}
                  className="rounded object-cover"
                />
                <div className="flex flex-col">
                  <h1 className="font-semibold text-lg">Languages</h1>
                  <p className="text-sm text-gray-500">
                    Spanish, English, Russian
                  </p>
                </div>
              </div>
              <Image
                src="/front_arrow.png"
                alt="front_arrow"
                width={20}
                height={20}
                className="opacity-70"
              />
            </div>
            <hr />
            <div className="flex items-center justify-between py-2 my-3 hover:bg-gray-500/10 px-5">
              <div className="flex items-center gap-4">
                <Image
                  src="/gray_parental_control.png"
                  alt="profile_image"
                  width={25}
                  height={48}
                  className="rounded object-cover"
                />
                <div className="flex flex-col">
                  <h1 className="font-semibold text-lg">
                    Adjust parental controls
                  </h1>
                  <p className="text-sm text-gray-500">
                    Edit age ratings and title restrictions
                  </p>
                </div>
              </div>
              <Image
                src="/front_arrow.png"
                alt="front_arrow"
                width={20}
                height={20}
                className="opacity-70"
              />
            </div>
            <hr />
            <div className="flex items-center justify-between py-2 my-3 hover:bg-gray-500/10 px-5">
              <div className="flex items-center gap-4">
                <Image
                  src="/gray_subtitles.png"
                  alt="profile_image"
                  width={25}
                  height={48}
                  className="rounded object-cover"
                />
                <div className="flex flex-col">
                  <h1 className="font-semibold text-lg">
                    Appearance of subtitles
                  </h1>
                  <p className="text-sm text-gray-500">
                    Customize the appearance of subtitles
                  </p>
                </div>
              </div>
              <Image
                src="/front_arrow.png"
                alt="front_arrow"
                width={20}
                height={20}
                className="opacity-70"
              />
            </div>
            <hr />
            <div className="flex items-center justify-between py-2 my-3 hover:bg-gray-500/10 px-5">
              <div className="flex items-center gap-4">
                <Image
                  src="/gray_play.png"
                  alt="profile_image"
                  width={25}
                  height={48}
                  className="rounded object-cover"
                />
                <div className="flex flex-col">
                  <h1 className="font-semibold text-lg">Playback settings</h1>
                  <p className="text-sm text-gray-500">
                    Configure autoplay, and audio and video quality
                  </p>
                </div>
              </div>
              <Image
                src="/front_arrow.png"
                alt="front_arrow"
                width={20}
                height={20}
                className="opacity-70"
              />
            </div>
            <hr />
            <div className="flex items-center justify-between py-2 my-3 hover:bg-gray-500/10 px-5">
              <div className="flex items-center gap-4">
                <Image
                  src="/notifications.png"
                  alt="profile_image"
                  width={25}
                  height={48}
                  className="rounded object-cover"
                />
                <div className="flex flex-col">
                  <h1 className="font-semibold text-lg">
                    Notification settings
                  </h1>
                  <p className="text-sm text-gray-500">
                    Manage automatic notifications, email notifications, and SMS
                    messages
                  </p>
                </div>
              </div>
              <Image
                src="/front_arrow.png"
                alt="front_arrow"
                width={20}
                height={20}
                className="opacity-70"
              />
            </div>
            <hr />
            <div className="flex items-center justify-between py-2 my-3 hover:bg-gray-500/10 px-5">
              <div className="flex items-center gap-4">
                <Image
                  src="/gray_history.png"
                  alt="profile_image"
                  width={25}
                  height={48}
                  className="rounded object-cover"
                />
                <div className="flex flex-col">
                  <h1 className="font-semibold text-lg">Viewing activity</h1>
                  <p className="text-sm text-gray-500">
                    Manage viewing history and ratings
                  </p>
                </div>
              </div>
              <Image
                src="/front_arrow.png"
                alt="front_arrow"
                width={20}
                height={20}
                className="opacity-70"
              />
            </div>
            <hr />
            <div className="flex items-center justify-between py-2 my-3 hover:bg-gray-500/10 px-5">
              <div className="flex items-center gap-4">
                <Image
                  src="/gray_security.png"
                  alt="profile_image"
                  width={25}
                  height={48}
                  className="rounded object-cover"
                />
                <div className="flex flex-col">
                  <h1 className="font-semibold text-lg">
                    Data and privacy settings
                  </h1>
                  <p className="text-sm text-gray-500">
                    Manage the use of your personal information
                  </p>
                </div>
              </div>
              <Image
                src="/front_arrow.png"
                alt="front_arrow"
                width={20}
                height={20}
                className="opacity-70"
              />
            </div>
          </div>
          <button className="w-full border border-gray-500/50 rounded flex flex-row items-center justify-center py-4 my-7 font-bold text-gray-900/60 hover:bg-gray-900/20">
            <Image
              src="/gray_delete.png"
              alt="delete_icon"
              width={25}
              height={25}
              className="mr-2"
            />
            Delete profile
          </button>
          {isMainProfile && (
            <p className="text-sm text-gray-500 mb-10 text-center">
              The main profile cannot be deleted.
            </p>
          )}
        </div>
      </section>
      <footer className="text-gray-800 py-15 px-40 bg-gray-500/10">
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
