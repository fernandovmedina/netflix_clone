import { pageMetadata } from "@/utils/metadata";
export const metadata = pageMetadata("Choose a Profile", "Choose who is watching Netflix Clone.", { authenticated: true });
export default function Layout({ children }: { children: React.ReactNode }) { return children; }
