"use client";

type ReviewDetailErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function ReviewDetailError({ error, reset }: ReviewDetailErrorProps) {
  return (
    <section
      role="alert"
      style={{
        padding: "20px",
        border: "1px solid #fecaca",
        borderRadius: "18px",
        background: "#fef2f2",
      }}
    >
      <h2 style={{ margin: 0, fontSize: "1.5rem" }}>Unable to load this review</h2>
      <p style={{ margin: "10px 0 0", color: "#7f1d1d" }}>{error.message}</p>
      <button
        type="button"
        onClick={reset}
        style={{
          marginTop: "16px",
          padding: "10px 14px",
          border: 0,
          borderRadius: "10px",
          background: "#b91c1c",
          color: "#fff",
          cursor: "pointer",
        }}
      >
        Try again
      </button>
    </section>
  );
}
