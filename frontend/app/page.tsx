"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";

export default function Home() {
  const [email, setEmail] = useState<string>("");

  const router: any = useRouter();

  const carrouselData = [
    { id: 1, imgSrc: "/home/stranger_things.webp" },
    { id: 2, imgSrc: "/home/frankenstin.webp" },
    { id: 3, imgSrc: "/home/kpop.webp" },
    { id: 4, imgSrc: "/home/mariposa_de_barrio.webp" },
    { id: 5, imgSrc: "/home/juan_gabriel.webp" },
    { id: 6, imgSrc: "/home/troll2.webp" },
    { id: 7, imgSrc: "/home/it.webp" },
    { id: 8, imgSrc: "/home/wednesday.webp" },
    { id: 9, imgSrc: "/home/fugue.webp" },
    { id: 10, imgSrc: "/home/the_believers.webp" },
  ];

  const Carrousel = ({ items }: any) => {
    const carouselRef: any = useRef(null);

    const scrollLeft = () => {
      if (!carouselRef.current) return;
      const width = carouselRef.current.clientWidth;
      carouselRef.current.scrollBy({ left: -width, behavior: "smooth" });
    };

    const scrollRight = () => {
      if (!carouselRef.current) return;
      const width = carouselRef.current.clientWidth;
      carouselRef.current.scrollBy({ left: width, behavior: "smooth" });
    };

    return (
      <div className="relative w-full pb-10">
        <button
          onClick={scrollLeft}
          className="absolute left-0 top-1/2 -translate-y-1/2 z-10 bg-black/50 hover:bg-black p-3 rounded-full"
        >
          <span className="text-white text-2xl">‹</span>
        </button>

        <div
          ref={carouselRef}
          className="flex overflow-x-auto no-scrollbar scroll-smooth snap-x snap-mandatory"
        >
          {items.map((item: any) => (
            <div key={item.id} className="w-1/5 shrink-0 px-2 snap-start">
              <Image
                src={item.imgSrc}
                alt="carousel item"
                width={197}
                height={276}
                className="h-auto w-[220px] rounded-xl hover:w-80"
              />
            </div>
          ))}
        </div>

        <button
          onClick={scrollRight}
          className="absolute right-0 top-1/2 -translate-y-1/2 z-10 bg-black/50 hover:bg-black p-3 rounded-full"
        >
          <span className="text-white text-2xl">›</span>
        </button>
      </div>
    );
  };

  const SliderCards = () => {
    return (
      <div className="pb-10">
        <div className="grid grid-cols-4 gap-3">
          <div className="bg-linear-to-br from-indigo-900 px-3 py-5 to-purple-950 rounded-xl">
            <h1 className="font-extrabold text-2xl">Enjoy on your TV</h1>
            <p className="text-gray-400 mt-3">
              Watch on Smart TVs, Playstation, Xbox, Chromecast, Apple TV,
              Blu-ray players, and more.
            </p>
          </div>
          <div className="bg-linear-to-br from-indigo-900 px-3 py-5 to-purple-950 rounded-xl">
            <h1 className="font-extrabold text-2xl">
              Download your shows to watch offline
            </h1>
            <p className="text-gray-400 mt-3">
              Save your favorites easily and always have something to watch.
            </p>
          </div>
          <div className="bg-linear-to-br from-indigo-900 px-3 py-5 to-purple-950 rounded-xl">
            <h1 className="font-extrabold text-2xl">Watch everywhere</h1>
            <p className="text-gray-400 mt-3">
              Stream unlimited movies and TV shows on your phone, tablet,
              laptop, and TV.
            </p>
          </div>
          <div className="bg-linear-to-br from-indigo-900 px-3 py-5 to-purple-950 rounded-xl">
            <h1 className="font-extrabold text-2xl">
              Create profiles for kids
            </h1>
            <p className="text-gray-400 mt-3">
              Send kids on adventures with their favorite characters in a space
              made just for them — free with your membership.
            </p>
          </div>
        </div>
      </div>
    );
  };

  const FrequentlyAskedQuestions = () => {
    const frequentlyAskedQuestionsData = 
    [
      { id: 1, title: "What is Netflix", text: "Netflix is a streaming service that offers a wide variety of award-winning TV shows, movies, anime, documentaries, and more on thousands of internet-connected devices. You can watch as much as you want, whenever you want – all for one low monthly price. There's always something new to discover and new TV shows and movies are added every week!", },
      { id: 2, title: "How much does Netflix cost?", text: "Watch Netflix on your smartphone, tablet, Smart TV, laptop, or streaming device, all for one fixed monthly fee. Plans range from MXN 119 to MXN 329/month.", },
      { id: 3, title: "Where can i watch?", text: "Watch anywhere, anytime. Sign in with your Netflix account to watch instantly on the web at netflix.com from your personal computer or on any internet-connected device that offers the Netflix app, including smart TVs, smartphones, tablets, streaming media players and game consoles. You can also download your favorite shows with the iOS or Android app. Use downloads to watch while you're on the go and without an internet connection. Take Netflix with you anywhere.", },
      { id: 4, title: "How do i cancel?", text: "Netflix is flexible. You can easily cancel your account online in two clicks. There are no cancellation fees – start or stop your account anytime.", },
      { id: 5, title: "What can i watch on Netflix?", text: "Netflix has an extensive library of feature films, documentaries, TV shows, anime, award-winning Netflix originals, and more. Watch as much as you want, anytime you want.", },
      { id: 6, title: "Is netflix good for kids?", text: "The Netflix Kids experience is included in your membership to give parents control while kids enjoy family-friendly TV shows and movies in their own space. Kids profiles come with PIN-protected parental controls that let you restrict the maturity rating of content kids can watch and block specific titles you don’t want kids to see.", },
    ];

    const [openId, setOpenId] = useState<number | null>(null);

    const toggle = (id: number) => {
      setOpenId(openId === id ? null : id);
    };

    return (
      <div>
        {frequentlyAskedQuestionsData.map((item) => (
          <div key={item.id}>
            <div
              onClick={() => toggle(item.id)}
              className="flex flex-row items-center justify-between bg-gray-800 my-3 px-5 py-3 hover:bg-gray-700 hover:cursor-pointer"
            >
              <h1 className="font-bold text-2xl">{item.title}</h1>
              <Image src="/home/more.png" alt="expand_icon" width={50} height={50} className="h-auto" />
            </div>
            {openId === item.id && (
              <div className="bg-gray-800 -mt-2 px-2 py-2">
                <h1>{item.text}</h1>
              </div>
            )}
          </div>
        ))}
      </div>
    );
  };

  const verifyEmail = () => {
    const emailInput: HTMLElement | any = document.getElementById("email_input");

    if (email === "") {
      emailInput.focus();
    }

    if (email !== "") {
      localStorage.setItem("signup_email", email);
      router.push("/signup/linkRegistration");
    }
  }

  return (
    <main className="bg-black">
      <div className="relative bg-[url(/home/hero.jpg)] bg-cover bg-center px-36 py-5">
        <div className="absolute inset-0 bg-black/70"></div>
        <div className="relative">
          <nav className="flex flex-row w-full items-center pt-2">
            <div className="w-1/2">
              <Link href="/">
                <Image
                  src="/netflix_logo.svg"
                  alt="netflix_logo"
                  width={150}
                  height={41}
                  className="h-auto"
                />
              </Link>
            </div>
            <div className="w-1/2 flex flex-row items-stretch justify-end">
              <div className="flex flex-row border-2 border-gray-400 bg-black/60 px-2 rounded-lg mr-3 h-full">
                <Image
                  src="/language.png"
                  alt="language_icon"
                  width={35}
                  height={35}
                  className="h-auto"
                />
                <select className="text-white pl-2 bg-transparent font-semibold">
                  <option>English</option>
                  <option>Español</option>
                </select>
              </div>

              <Link
                href="/login"
                className="text-white font-bold bg-red-700 rounded px-4 flex items-center"
              >
                Sign In
              </Link>
            </div>
          </nav>
          <div className="pt-40 text-white flex flex-col justify-center items-center mx-auto max-w-[800px] pb-48">
            <h1 className="text-6xl font-extrabold text-center">
              Unlimited movies, TV shows, and more
            </h1>
            <h2 className="text-xl font-extrabold mt-5 mb-7 text-center">
              Starts at MXN 119. Cancel anytime
            </h2>
            <p className="font-bold mb-3 text-center">
              Ready to watch? Enter your email to create or restart your
              membership.
            </p>
            <div className="flex flex-row items-stretch">
              <input
                id="email_input"
                className="bg-transparent border-2 font-bold border-gray-600 rounded px-3 h-full py-3 w-[400px] text-white placeholder:text-gray-400"
                onChange={e => setEmail(e.target.value)}
                placeholder="Email address"
              />

              <button onClick={verifyEmail} className="flex flex-row font-extrabold items-center bg-red-700 hover:bg-red-500 hover:cursor-pointer px-4 rounded ml-3 h-full py-3">
                Get Started
                <Image
                  src="/home/greater.png"
                  alt="greater_icon"
                  width={20}
                  height={20}
                  className="ml-2 h-auto"
                />
              </button>
            </div>
          </div>
        </div>
      </div>
      <div className="bg-black text-white px-28 relative">
        <div className="flex flex-row items-center w-full pt-10 pb-16 relative z-10">
          <Image
            src="/home/popcorn.png"
            alt="popcorn_image"
            width={70}
            height={70}
            className="h-auto"
          />

          <div className="flex flex-row items-center justify-between bg-linear-to-r from-purple-900 via-purple-700 to-sky-900 w-full px-10 py-4 rounded-xl ml-5">
            <div className="flex flex-col">
              <h1 className="font-extrabold text-xl">
                The Netflix you love for just MXN 119.
              </h1>
              <p className="font-bold text-sm">
                Get our most affordable, ad-supported plan.
              </p>
            </div>

            <a className="bg-gray-800 px-3 py-2 rounded-md font-semibold cursor-pointer">
              Learn More
            </a>
          </div>
        </div>

        <h2 className="font-bold text-2xl pb-7">Trending Now</h2>
        <Carrousel items={carrouselData} />

        <h2 className="font-bold text-2xl pb-7">More Reasons to Join</h2>
        <SliderCards />

        <h2 className="font-bold text-2xl pb-7">Frequently Asked Questions</h2>
        <FrequentlyAskedQuestions />

        <div className="flex flex-col justify-center text-center mt-20 mb-32">
          <h2 className="font-bold mb-5">Ready to watch? Enter your email to create or restart your membership.</h2>
          <div className="flex flex-row items-stretch justify-center">
            <input
              className="bg-transparent border-2 font-bold border-gray-600 rounded px-3 h-full py-3 w-[400px] text-white placeholder:text-gray-400"
              placeholder="Email address"
            />

            <button className="flex flex-row font-extrabold items-center bg-red-700 px-4 rounded ml-3 h-full py-3">
              Get Started
              <Image
                src="/home/greater.png"
                alt="greater_icon"
                width={20}
                height={20}
                className="ml-2 h-auto"
              />
            </button>
          </div>
        </div>

        <footer className="mb-48">
          <h1 className="font-bold mb-5">Questions? Call <a href="tel:8008539947" className="underline">800 953 9947</a></h1>
          <div className="flex flex-row w-full text-sm underline text-gray-400">
            <div className="w-1/4 flex flex-col">
              <a href="" className="hover:text-gray-200 mb-3">FAQ</a>
              <a href="" className="hover:text-gray-200 mb-3">Investor Relations</a>
              <a href="" className="hover:text-gray-200 mb-3">Buy Gift Cards</a>
              <a href="" className="hover:text-gray-200 mb-3">Cookie Preferences</a>
              <a href="" className="hover:text-gray-200 mb-3">Legal Notices</a>
            </div>
            <div className="w-1/4 flex flex-col">
              <a href="" className="hover:text-gray-200 mb-3">Help Center</a>
              <a href="" className="hover:text-gray-200 mb-3">Jobs</a>
              <a href="" className="hover:text-gray-200 mb-3">Ways to Watch</a>
              <a href="" className="hover:text-gray-200 mb-3">Corporate Information</a>
              <a href="" className="hover:text-gray-200 mb-3">Only on Netflix</a>
            </div>
            <div className="w-1/4 flex flex-col">
              <a href="" className="hover:text-gray-200 mb-3">Account</a>
              <a href="" className="hover:text-gray-200 mb-3">Netflix Shop</a>
              <a href="" className="hover:text-gray-200 mb-3">Terms of Use</a>
              <a href="" className="hover:text-gray-200 mb-3">Contact Us</a>
            </div>
            <div className="w-1/4 flex flex-col">
              <a href="" className="hover:text-gray-200 mb-3">Media Center</a>
              <a href="" className="hover:text-gray-200 mb-3">Reddem Gift Cards</a>
              <a href="" className="hover:text-gray-200 mb-3">Privacy</a>
              <a href="" className="hover:text-gray-200 mb-3">Speed Test</a>
            </div>
          </div>
          <div className="flex flex-row border-2 border-gray-400 bg-black/60 px-2 rounded-lg w-36">
            <Image
              src="/language.png"
              alt="language_icon"
              width={35}
              height={35}
              className="h-auto"
            />
            <select className="text-white pl-2 bg-transparent font-semibold">
              <option>English</option>
              <option>Español</option>
            </select>
          </div>
          <h1 className="text-gray-200 mt-5 text-xl">Netflix</h1>
          <h1 className="text-sm font-extrabold mt-3">Made by <a target="_blank" href="https://github.com/fernandovmedina" className="underline">@fernandovmedina</a> & <a className="underline" href="https://neurovix.com.mx" target="_blank">@Neurovix</a></h1>
        </footer>
      </div>
    </main>
  );
}
