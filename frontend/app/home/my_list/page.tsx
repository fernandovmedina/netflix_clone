import { CatalogPage } from "@/components/CatalogPage";
import { pageMetadata } from "@/utils/metadata";

export const metadata = pageMetadata("My List", "Return to the movies and series saved to your list.", { authenticated: true });

export default function MyList() {
  return <CatalogPage mode="favorites" />;
}
