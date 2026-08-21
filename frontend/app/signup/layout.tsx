import type { Metadata } from "next";
import "../globals.css";

export const metadata: Metadata = {
  title: "Netflix",
  description: "Created by @github.com/fernandovmedina",
};

export default function SignUpLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="font-sans antialiased">
      {children}
    </div>
  );
}
