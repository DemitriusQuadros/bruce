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
        let message = `HTTP ${response.status}`;
        try {
            const data = await response.json();
            if (data && data.error) {
                message = data.error;
            } else if (data && data.message) {
                message = data.message;
            }
        } catch (_) {
            const text = await response.text();
            if (text) message = text;
        }
        throw new ApiError(message, response.status);
    }

    if (response.status === 204) {
        return null;
    }

    return response.json();
}
