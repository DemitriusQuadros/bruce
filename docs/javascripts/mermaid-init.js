// Initialize Mermaid.js for Material for MkDocs with support for instant navigation and theme switching
function initMermaid() {
  if (typeof mermaid === "undefined") return;

  const isDark = document.body.getAttribute("data-md-color-scheme") === "slate";

  mermaid.initialize({
    startOnLoad: false,
    theme: isDark ? "dark" : "base",
    securityLevel: "loose",
    fontFamily: "Roboto, -apple-system, BlinkMacSystemFont, sans-serif",
    themeVariables: isDark ? {
      darkMode: true,
      background: "transparent",
      mainBkg: "#111827",
      primaryColor: "#1e293b",
      primaryTextColor: "#f8fafc",
      primaryBorderColor: "#38bdf8",
      lineColor: "#94a3b8",
      secondaryColor: "#111827",
      tertiaryColor: "#1e293b",
      clusterBkg: "#0f172a",
      clusterBorder: "rgba(255, 255, 255, 0.12)",
      edgeLabelBackground: "#0b0f19",
      nodeBorder: "#38bdf8",
      nodeTextColor: "#f8fafc",
      defaultLinkColor: "#94a3b8",
      titleColor: "#f8fafc"
    } : {
      darkMode: false,
      background: "transparent",
      mainBkg: "#ffffff",
      primaryColor: "#ffffff",
      primaryTextColor: "#262320",
      primaryBorderColor: "#d8cfbf",
      lineColor: "#665e53",
      secondaryColor: "#ede7db",
      tertiaryColor: "#faf7f2",
      clusterBkg: "#ede7db",
      clusterBorder: "#d8cfbf",
      edgeLabelBackground: "#f7f3eb",
      nodeBorder: "#d8cfbf",
      nodeTextColor: "#262320",
      defaultLinkColor: "#665e53",
      titleColor: "#262320"
    }
  });

  const nodes = document.querySelectorAll("pre.mermaid, div.mermaid, .mermaid");
  if (!nodes.length) return;

  nodes.forEach(el => {
    // Save original source text if not yet saved
    if (!el.dataset.mermaidSource) {
      const code = el.querySelector("code");
      el.dataset.mermaidSource = code ? code.textContent : el.textContent;
    }
    // Restore raw code so mermaid can re-render cleanly
    el.removeAttribute("data-processed");
    el.innerHTML = el.dataset.mermaidSource;
  });

  mermaid.run({
    nodes: document.querySelectorAll("pre.mermaid, div.mermaid, .mermaid")
  });
}

// Initial page load
document.addEventListener("DOMContentLoaded", () => {
  initMermaid();

  // Watch for palette toggle changes (e.g. switching between light paper and dark slate)
  const observer = new MutationObserver(mutations => {
    for (const mutation of mutations) {
      if (mutation.type === "attributes" && mutation.attributeName === "data-md-color-scheme") {
        initMermaid();
      }
    }
  });
  observer.observe(document.body, { attributes: true, attributeFilter: ["data-md-color-scheme"] });
});

// Re-render when page content changes via Material for MkDocs instant loading
if (typeof document$ !== "undefined") {
  document$.subscribe(() => {
    initMermaid();
  });
}
