export default function QueueLoading() {
  return (
    <section aria-busy="true" role="status">
      <h2 style={{ margin: 0, fontSize: "1.5rem" }}>Loading review queue</h2>
      <p style={{ margin: "8px 0 0", color: "#475569" }}>
        Fetching the latest pending review work from the backend.
      </p>
    </section>
  );
}
