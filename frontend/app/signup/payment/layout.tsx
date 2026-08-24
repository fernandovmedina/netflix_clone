import { pageMetadata } from "@/utils/metadata";
export const metadata = {
  ...pageMetadata("Choose a Payment Method", "Select how you want to pay for your Netflix Clone plan.", { social: true }),
  title: { default: "Choose a Payment Method", template: "%s · Netflix Clone" },
};
export default function Layout({ children }: { children: React.ReactNode }) { return children; }
