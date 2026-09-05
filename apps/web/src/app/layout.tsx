import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Taskline",
  description:
    "Issue tracking and project management application built with Go, PostgreSQL and Next.js.",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body className="antialiased">{children}</body>
    </html>
  );
}
