import type { Metadata } from "next";
import { pageMetadata } from "@/utils/metadata";
import "../globals.css";

export const metadata: Metadata = {
  ...pageMetadata("Create Your Account", "Start your Netflix Clone membership in a few simple steps.", { social: true }),
  title: { default: "Create Your Account", template: "%s · Netflix Clone" },
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
