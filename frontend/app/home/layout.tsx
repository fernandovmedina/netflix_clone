import { HomeGuard } from "@/components/HomeGuard";
import { pageMetadata } from "@/utils/metadata";

export const metadata = {
  ...pageMetadata("Home", "Browse personalized movies and series.", { authenticated: true }),
  title: { default: "Home", template: "%s · Netflix Clone" },
};

export default function HomeLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <HomeGuard>{children}</HomeGuard>;
}
