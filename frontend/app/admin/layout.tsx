import { AdminShell } from "@/components/admin/AdminShell";
import { pageMetadata } from "@/utils/metadata";

export const metadata = {
  ...pageMetadata("Admin Titles", "Manage the movie and series catalog.", { authenticated: true }),
  title: { default: "Admin Titles", template: "%s · Netflix Clone" },
};

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <AdminShell>{children}</AdminShell>;
}
