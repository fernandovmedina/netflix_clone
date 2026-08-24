import { CatalogPage } from "@/components/CatalogPage";
import { pageMetadata } from "@/utils/metadata";

export const metadata = pageMetadata("Movies", "Explore movies available to watch now.", { authenticated: true });

export default function Movies() {
  return <CatalogPage mode="movies" />;
}
