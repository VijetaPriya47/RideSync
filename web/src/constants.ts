export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8081';

const getWsUrl = () => {
    if (process.env.NEXT_PUBLIC_WEBSOCKET_URL) return process.env.NEXT_PUBLIC_WEBSOCKET_URL;

    // Replace http with ws and https with wss
    const base = API_URL.replace(/^http/, 'ws');
    // Ensure the /ws prefix is included if not already present (matches API Gateway mux)
    return base.endsWith('/ws') ? base : `${base}/ws`;
};

export const WEBSOCKET_URL = getWsUrl();
