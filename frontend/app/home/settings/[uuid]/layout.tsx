import { pageMetadata } from "@/utils/metadata";
export const metadata = pageMetadata("Profile Settings", "Update the selected Netflix Clone profile.", { authenticated: true });
export default function Layout({ children }: { children: React.ReactNode }) { return children; }
