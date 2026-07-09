import { createClient } from "@/utils/supabase/client";
import { ArrowRightCircle, Edit2, Info, User } from "@deemlol/next-icons";
import Image from "next/image";
import { useEffect, useState } from "react";

export const Navbar = () => {
  const [uuid, setUuid] = useState<string | null>("");

  useEffect(() => {
    setUuid(new URLSearchParams(window.location.search).get("uuid"));
  }, []);

  const handleLogOut = async () => {
    const supabase = createClient();

    const { error } = await supabase.auth.signOut();

    if (error) {
      console.error("Error signing out: ", error.message);
    } else {
      window.location.href = "/";
    }
  }

  const ProfileView = ({ url, name }: { url: string; name: string }) => {
    return (
      <div className="flex flex-row items-center mb-3 gap-3">
        <Image src={url} alt={name} width={30} height={15} />
        <p className="text-sm hover:underline">{name}</p>
      </div>
    );
  };

  const ProfileDropdown = () => {
    const api_example = {
      profiles: [
        {
          id: 1,
          name: "Fernando",
          url: "https://occ-0-7553-114.1.nflxso.net/dnm/api/v6/vN7bi_My87NPKvsBoib006Llxzg/AAAABW7Wui3ZqHqBvl3R__TmY0sDZF-xBxJJinhVWRwu7OmYkF2bdwH4nqfnyT3YQ-JshQvap33bDbRLACSoadpKwbIQIBktdtHjxw.png?r=201",
        },
        {
          id: 2,
          name: "Mariajose",
          url: "https://occ-0-7553-114.1.nflxso.net/dnm/api/v6/vN7bi_My87NPKvsBoib006Llxzg/AAAABW7Wui3ZqHqBvl3R__TmY0sDZF-xBxJJinhVWRwu7OmYkF2bdwH4nqfnyT3YQ-JshQvap33bDbRLACSoadpKwbIQIBktdtHjxw.png?r=201",
        },
        {
          id: 3,
          name: "Norma",
          url: "https://occ-0-7553-114.1.nflxso.net/dnm/api/v6/vN7bi_My87NPKvsBoib006Llxzg/AAAABW7Wui3ZqHqBvl3R__TmY0sDZF-xBxJJinhVWRwu7OmYkF2bdwH4nqfnyT3YQ-JshQvap33bDbRLACSoadpKwbIQIBktdtHjxw.png?r=201",
        },
      ],
    };

    return (
      <div>
        {api_example.profiles.map((profile) => (
          <ProfileView key={profile.id} url={profile.url} name={profile.name} />
        ))}
        <div className="flex flex-row items-center my-3 gap-3 text-sm">
          <Edit2 size={24} color="#FFFFFF" />
          <a href="" className="hover:cursor-pointer hover:underline">
            Manage profiles
          </a>
        </div>
        <div className="flex flex-row items-center mb-3 gap-3 text-sm">
          <ArrowRightCircle size={24} color="#FFFFFF" />
          <a href="" className="hover:cursor-pointer hover:underline">
            Transfer profile
          </a>
        </div>
        <div className="flex flex-row items-center mb-3 gap-3 text-sm">
          <User size={24} color="#FFFFFF" />
          <a href="" className="hover:cursor-pointer hover:underline">
            Account
          </a>
        </div>
        <div className="flex flex-row items-center pb-3 gap-3 text-sm border-b border-gray-400">
          <Info size={24} color="#FFFFFF" />
          <a href="" className="hover:cursor-pointer hover:underline">
            Help center
          </a>
        </div>
        <div className="flex flex-row items-center hover:font-bold text-sm">
          <button onClick={handleLogOut} className="hover:cursor-pointer hover:underline">
            Log out
          </button>
        </div>
      </div>
    );
  };

  return (
    <nav className="text-white flex flex-row items-center px-14 py-4 w-full">
      <div className="w-1/12">
        <a href={`/home?uuid=${uuid}`}>
          <Image
            src="/netflix_logo.svg"
            alt="netflix_logo"
            width={100}
            height={27}
            className="h-auto"
          />
        </a>
      </div>
      <div className="flex flex-row items-center w-9/12 justify-start px-7 gap-5 text-sm">
        <a href={`/home?uuid=${uuid}`} className="hover:text-gray-400/90">
          Home
        </a>
        <a
          href={`/home/series?uuid=${uuid}`}
          className="hover:text-gray-400/90"
        >
          Series
        </a>
        <a href={`/home/movies?uuid=${uuid}`} className="hover:text-gray-400/90">
          Movies
        </a>
        <a href={`/home/new_arrivals?uuid=${uuid}`} className="hover:text-gray-400/90">
          Popular new arrivals
        </a>
        <a href={`/home/my_list?uuid=${uuid}`} className="hover:text-gray-400/90">
          My List
        </a>
      </div>
      <div className="flex flex-row items-center w-2/12 gap-3">
        <Image
          src="/white_search.png"
          alt="white_search_icon"
          width={30}
          height={15}
        />
        <a href="" className="text-sm">
          Children
        </a>
        <Image
          src="/white_notifications.png"
          alt="white_notifications"
          width={30}
          height={15}
        />
        <div className="relative group flex items-center gap-1">
          <div className="flex items-center gap-1">
            <Image
              src="https://occ-0-7553-114.1.nflxso.net/dnm/api/v6/vN7bi_My87NPKvsBoib006Llxzg/AAAABW7Wui3ZqHqBvl3R__TmY0sDZF-xBxJJinhVWRwu7OmYkF2bdwH4nqfnyT3YQ-JshQvap33bDbRLACSoadpKwbIQIBktdtHjxw.png?r=201"
              alt="profile_image"
              width={30}
              height={15}
            />
            <Image
              src="/white_dropdown.png"
              alt="white_dropdown_icon"
              width={15}
              height={15}
            />
          </div>
          <div className="absolute left-0 right-0 top-full h-3"></div>
          <div className="absolute right-0 top-full mt-3 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-opacity duration-150 bg-black/90 rounded shadow-lg p-3 min-w-[200px] z-50">
            <ProfileDropdown />
          </div>
        </div>
      </div>
    </nav>
  );
};
