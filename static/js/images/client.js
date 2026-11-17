const isPrefixed = window.location.pathname.startsWith('/presenter');
const APP_BASE_PATH = isPrefixed ? '/presenter' : '';

export function withBasePath(path) {
    return APP_BASE_PATH + path;
}

export function getAuthHeader() {
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
        throw new Error(body.error || `Upload de imagem falhou (status ${res.status})`);
    }
    return res.json();
}

export async function fetchImageList() {
    const res = await fetch(withBasePath('/api/images'));
    if (!res.ok) {
        throw new Error(`Falha ao listar imagens (status ${res.status})`);
    }
    const body = await res.json();
    return body.images || [];
}

export async function uploadVideo(file) {
    const formData = new FormData();
    formData.append('files', file);
    const res = await fetch(withBasePath('/api/videos'), {
        method: 'POST',
        headers: { Authorization: getAuthHeader() },
        body: formData
    });
    if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `Upload de vídeo falhou (status ${res.status})`);
    }
    return res.json();
}

export async function fetchVideoList() {
    const res = await fetch(withBasePath('/api/videos'));
    if (!res.ok) {
        throw new Error(`Falha ao listar vídeos (status ${res.status})`);
    }
    const body = await res.json();
    return body.videos || [];
}

export async function deleteVideo(name) {
    const res = await fetch(withBasePath(`/api/videos/${encodeURIComponent(name)}`), {
        method: 'DELETE',
        headers: { Authorization: getAuthHeader() }
    });
    if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `Falha ao remover vídeo (status ${res.status})`);
    }
    return res.json();
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

export function emitVideoContent(name) {
    return fetch(withBasePath('/api/content/set/main'), {
        method: 'POST',
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ content: name, type: 'VIDEO' })
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

export async function fetchCurrentContent() {
    const res = await fetch(withBasePath('/api/content?providerId=main'));
    if (!res.ok) {
        return null;
    }
    const data = await res.json();
    return data.main || null;
}

export function sendVideoCommand(command, content = '') {
    return fetch(withBasePath('/api/content/set/command'), {
        method: 'POST',
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            contentId: `${command}-${Date.now()}`,
            type: 'COMMAND',
            content
        })
    });
}

export function playVideo() {
    return sendVideoCommand('videoPlay');
}

export function pauseVideo() {
    return sendVideoCommand('videoPause');
}

export function forwardVideo(seconds = 10) {
    return sendVideoCommand('videoForward', String(seconds));
}

export function rewindVideo(seconds = 10) {
    return sendVideoCommand('videoRewind', String(seconds));
}

export function restartVideo() {
    return sendVideoCommand('videoRestart');
}

export async function fetchDownloaderStatus() {
    const res = await fetch(withBasePath('/api/videos/downloader/status'));
    if (!res.ok) {
        return { available: false, message: 'Falha ao verificar status das ferramentas.' };
    }
    return res.json();
}

export async function downloadVideoFromUrl(url) {
    const res = await fetch(withBasePath('/api/videos/download'), {
        method: 'POST',
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ url })
    });
    const body = await res.json().catch(() => ({}));
    if (!res.ok) {
        throw new Error(body.error || `Download falhou (status ${res.status})`);
    }
    return body;
}

export async function downloadVideoWithProgress(url, onProgress) {
    const res = await fetch(withBasePath('/api/videos/download'), {
        method: 'POST',
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json',
            'Accept': 'text/event-stream'
        },
        body: JSON.stringify({ url })
    });

    if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `Download falhou (status ${res.status})`);
    }

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let finalName = '';

    while (true) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const parts = buffer.split('\n\n');
        buffer = parts.pop(); // Mantém o pedaço incompleto no buffer

        for (const part of parts) {
            const line = part.trim();
            if (!line.startsWith('data:')) continue;
            const jsonStr = line.replace(/^data:\s*/, '');
            try {
                const data = JSON.parse(jsonStr);
                if (data.status === 'error') {
                    throw new Error(data.message || data.error || 'Erro no download');
                }
                if (data.name) {
                    finalName = data.name;
                }
                if (onProgress) {
                    onProgress(data);
                }
            } catch (err) {
                if (err.message && !err.message.startsWith('Unexpected')) {
                    throw err;
                }
            }
        }
    }

    if (!finalName) {
        throw new Error('Não foi possível identificar o vídeo baixado.');
    }
    return { name: finalName };
}

export function thumbUrl(name) {
    return withBasePath(`/api/images/thumb?name=${encodeURIComponent(name)}`);
}

export function videoContentUrl(name) {
    return withBasePath(`/api/videos/content?name=${encodeURIComponent(name)}`);
}
