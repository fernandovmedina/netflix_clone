import Image from "next/image";
import Link from "next/link";

export function PaymentShell({ children }: { children: React.ReactNode }) {
  return <main className="min-h-dvh overflow-x-hidden bg-white text-black">
    <nav className="flex min-h-20 items-center justify-between border-b border-gray-300 px-4 py-5 sm:px-8 lg:px-10">
      <Link href="/" aria-label="Netflix home"><Image src="/netflix_logo.svg" alt="Netflix" width={160} height={43} className="h-auto w-28 sm:w-40" priority /></Link>
      <Link href="/login" className="text-base font-bold hover:underline sm:text-xl">Sign Out</Link>
    </nav>
    {children}
    <footer className="bg-gray-100 px-5 py-10 text-sm text-gray-700 sm:px-10 lg:px-[max(2.5rem,10vw)]">
      <a href="tel:8009539947">Questions? Call 800 953 9947 (Toll-Free)</a>
      <div className="mt-6 grid grid-cols-2 gap-x-6 gap-y-3 underline sm:grid-cols-4">
        <a href="#">FAQ</a><a href="#">Help Center</a><a href="#">Privacy</a><a href="#">Terms of Use</a>
        <a href="#">Cookie Preferences</a><a href="#">Corporate Information</a>
      </div>
    </footer>
  </main>;
}

export function CheckoutColumn({ children, wide = false }: { children: React.ReactNode; wide?: boolean }) {
  return <section className="mx-auto w-full px-5 py-10 sm:px-8 sm:py-14">
    <div className={`mx-auto w-full ${wide ? "max-w-6xl" : "max-w-xl"}`}>{children}</div>
  </section>;
}
