/* HTTP Client */

export class ApiError extends Error {
    constructor(message, status) {
        super(message);
        this.status = status;
        this.name = 'ApiError';
    }
}

const API = '';

export async function req(method, path, body) {
    const headers = { 'Content-Type': 'application/json' };
    const response = await fetch(API + path, {
        method,
        headers,
        body: body ? JSON.stringify(body) : null,
    });

    if (!response.ok) {
        const text = await response.text();
        throw new ApiError(text || `HTTP ${response.status}`, response.status);
    }

    if (response.status === 204) {
        return null;
    }

    return response.json();
}
