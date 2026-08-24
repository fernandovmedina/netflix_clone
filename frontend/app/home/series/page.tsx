import { CatalogPage } from "@/components/CatalogPage";
import { pageMetadata } from "@/utils/metadata";

export const metadata = pageMetadata("Series", "Discover series and episodes available to stream.", { authenticated: true });

export default function Series() {
  return <CatalogPage mode="series" />;
}
