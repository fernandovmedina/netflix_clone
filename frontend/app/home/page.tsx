import { CatalogPage } from "@/components/CatalogPage";
import { pageMetadata } from "@/utils/metadata";

export const metadata = pageMetadata("Home", "Browse personalized movies and series.", { authenticated: true });

export default function Home() {
  return <CatalogPage mode="home" />;
}
