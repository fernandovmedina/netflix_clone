import type { Metadata } from "next";
import { Providers } from "./providers";
import "./globals.css";

export const metadata: Metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000"),
  title: { default: "Netflix Clone — Movies, Series and More", template: "%s · Netflix Clone" },
  description: "Watch movies and series, discover new favorites, and manage your Netflix Clone account.",
  icons: { icon: [{ url: "/netflix.png", type: "image/png" }] },
  openGraph: { title: "Netflix Clone — Movies, Series and More", description: "Watch movies and series, discover new favorites, and manage your Netflix Clone account.", siteName: "Netflix Clone", type: "website" },
  twitter: { card: "summary_large_image", title: "Netflix Clone — Movies, Series and More", description: "Watch movies and series, discover new favorites, and manage your Netflix Clone account." },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="font-sans antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
