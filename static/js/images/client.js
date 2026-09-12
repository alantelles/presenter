function getAuthHeader() {
    const token = document.getElementById('basic-auth-token').value;
    return `Basic ${token}`;
}

export async function uploadImage(file) {
    const formData = new FormData();
    formData.append('files', file);
    const res = await fetch('/api/images', {
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
    const res = await fetch('/api/images');
    const body = await res.json();
    return body.images || [];
}

export function emitImageContent(name) {
    return fetch('/api/content/set/main', {
        method: 'POST',
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ content: name, type: 'IMAGE' })
    });
}

export function clearMainContent() {
    return fetch('/api/content/set/main', {
        method: 'POST',
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ content: '', type: 'TEXT' })
    });
}

export function thumbUrl(name) {
    return `/api/images/thumb?name=${encodeURIComponent(name)}`;
}
