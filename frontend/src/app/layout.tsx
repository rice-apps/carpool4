import type { Metadata } from "next";
import type { ReactNode } from "react";
import { Providers } from "./providers";
import "./globals.css";

// oxlint-disable-next-line react/only-export-components -- Next.js expects metadata beside the layout.
export const metadata: Metadata = {
  title: "Carpool | Rice rides together",
  description: "Find a seat or share a trip with Rice students.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
