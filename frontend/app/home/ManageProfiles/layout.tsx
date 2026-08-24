import { pageMetadata } from "@/utils/metadata";
export const metadata = pageMetadata("Manage Profiles", "Create and manage Netflix Clone viewing profiles.", { authenticated: true });
export default function Layout({ children }: { children: React.ReactNode }) { return children; }
