import type { Metadata } from "next";

export const siteName = "Netflix Clone";
export const siteDescription = "Watch movies and series, discover new favorites, and manage your Netflix Clone account.";

type PageMetadataOptions = { authenticated?: boolean; social?: boolean };

export function pageMetadata(title: string, description: string, options: PageMetadataOptions = {}): Metadata {
  const metadata: Metadata = { title, description };
  if (options.authenticated) {
    metadata.robots = { index: false, follow: false, nocache: true };
    metadata.openGraph = null;
    metadata.twitter = null;
  }
  if (options.social) {
    metadata.openGraph = { title, description, siteName, type: "website" };
    metadata.twitter = { card: "summary_large_image", title, description };
  }
  return metadata;
}
