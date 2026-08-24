import { CatalogPage } from "@/components/CatalogPage";
import { pageMetadata } from "@/utils/metadata";

export const metadata = pageMetadata("New Arrivals", "See the newest movies and series on Netflix Clone.", { authenticated: true });

export default function NewArrivals() {
  return <CatalogPage mode="new" />;
}
