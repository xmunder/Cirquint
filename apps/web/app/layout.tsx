import type { Metadata } from "next";
import type { PropsWithChildren } from "react";

import "./globals.css";

export const metadata: Metadata = {
  title: "Circuit Review Workbench",
  description: "Minimal review shell for circuit-review routes.",
};

export function ReviewShell({ children }: PropsWithChildren) {
  return (
    <div
      style={{
        minHeight: "100vh",
        padding: "32px 20px",
        background:
          "linear-gradient(180deg, rgba(255,255,255,1) 0%, rgba(244,246,251,1) 100%)",
      }}
    >
      <main
        style={{
          maxWidth: "960px",
          margin: "0 auto",
          padding: "32px",
          background: "#ffffff",
          border: "1px solid #dbe2f0",
          borderRadius: "20px",
          boxShadow: "0 18px 45px rgba(15, 23, 42, 0.08)",
        }}
      >
        <header style={{ marginBottom: "24px" }}>
          <p
            style={{
              margin: 0,
              fontSize: "0.85rem",
              fontWeight: 700,
              letterSpacing: "0.08em",
              textTransform: "uppercase",
              color: "#4f46e5",
            }}
          >
            Cirquint
          </p>
          <h1 style={{ margin: "8px 0 0", fontSize: "2rem" }}>
            Circuit review workbench
          </h1>
          <p style={{ margin: "12px 0 0", color: "#475569", lineHeight: 1.6 }}>
            Review queue and detail routes live inside this shell. The backend
            remains the source of truth.
          </p>
        </header>
        {children}
      </main>
    </div>
  );
}

export default function RootLayout({ children }: Readonly<PropsWithChildren>) {
  return (
    <html lang="en">
      <body>
        <ReviewShell>{children}</ReviewShell>
      </body>
    </html>
  );
}
