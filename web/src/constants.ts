const DEFAULT_API_URL = "http://localhost:8081";

/** Trailing slash or `/api` on NEXT_PUBLIC_API_URL would produce `//api/...` and break gateway public paths. */
function normalizeGatewayBase(raw: string | undefined): string {
  if (raw == null || raw.trim() === "") {
    return DEFAULT_API_URL;
  }
  let s = raw.trim();
  while (s.endsWith("/")) {
    s = s.slice(0, -1);
  }
  if (s.endsWith("/api")) {
    s = s.slice(0, -4);
    while (s.endsWith("/")) {
      s = s.slice(0, -1);
    }
  }
  return s.length > 0 ? s : DEFAULT_API_URL;
}

export const API_URL = normalizeGatewayBase(process.env.NEXT_PUBLIC_API_URL);

const getWsUrl = () => {
  const explicit = process.env.NEXT_PUBLIC_WEBSOCKET_URL?.trim();
  if (explicit) {
    let w = explicit;
    while (w.endsWith("/")) {
      w = w.slice(0, -1);
    }
    return w.endsWith("/ws") ? w : `${w}/ws`;
  }

  const base = API_URL.replace(/^http/, "ws");
  return base.endsWith("/ws") ? base : `${base}/ws`;
};

export const WEBSOCKET_URL = getWsUrl();
