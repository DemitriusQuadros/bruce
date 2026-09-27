// Initialize Mermaid.js for Material for MkDocs with support for instant navigation
function initMermaid() {
  if (typeof mermaid === "undefined") return;

  // Unwrap inner <code> so mermaid parses raw diagram text cleanly
  document.querySelectorAll("pre.mermaid").forEach(el => {
    const code = el.querySelector("code");
    if (code) {
      el.textContent = code.textContent;
    }
  });

  mermaid.initialize({
    startOnLoad: false,
    theme: "default",
    securityLevel: "loose",
    fontFamily: "Roboto, sans-serif"
  });

  mermaid.run({
    nodes: document.querySelectorAll("pre.mermaid, div.mermaid, .mermaid")
  });
}

// Initial page load
document.addEventListener("DOMContentLoaded", () => {
  initMermaid();
});

// Re-render when page content changes via Material for MkDocs instant loading
if (typeof document$ !== "undefined") {
  document$.subscribe(() => {
    initMermaid();
  });
}
