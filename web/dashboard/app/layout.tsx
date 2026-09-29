import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "OmniFleet Dispatcher",
  description: "Multi-tenant fleet tracking dashboard",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
