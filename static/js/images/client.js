const isPrefixed = window.location.pathname.startsWith('/presenter');
const APP_BASE_PATH = isPrefixed ? '/presenter' : '';

function withBasePath(path) {
    return APP_BASE_PATH + path;
}

function getAuthHeader() {
    const token = document.getElementById('basic-auth-token').value;
    return `Basic ${token}`;
}

export async function uploadImage(file) {
    const formData = new FormData();
    formData.append('files', file);
    const res = await fetch(withBasePath('/api/images'), {
        method: 'POST',
        headers: { Authorization: getAuthHeader() },
        body: formData
    });
    if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `upload failed with status ${res.status}`);
    }
    return res.json();
}

export async function fetchImageList() {
    const res = await fetch(withBasePath('/api/images'));
    if (!res.ok) {
        throw new Error(`failed to list images (status ${res.status})`);
    }
    const body = await res.json();
    return body.images || [];
}

export function emitImageContent(name) {
    return fetch(withBasePath('/api/content/set/main'), {
        method: 'POST',
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ content: name, type: 'IMAGE' })
    });
}

export function clearMainContent() {
    return fetch(withBasePath('/api/content/set/main'), {
        method: 'POST',
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ content: '', type: 'TEXT' })
    });
}

export function thumbUrl(name) {
    return withBasePath(`/api/images/thumb?name=${encodeURIComponent(name)}`);
}
