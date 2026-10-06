function getAuthHeader() {
    const token = document.getElementById('basic-auth-token').value;
    return `Basic ${token}`;
}

function authorizedRequest(path, bodyObj, method = 'POST') {
    return fetch(path, {
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(bodyObj),
        method
    });
}

function commandRequest(bodyObj) {
    return authorizedRequest('/api/content/set/command', bodyObj);
}

export function emitMainContent(content) {
    return authorizedRequest('/api/content/set/main', { content, type: 'TEXT' });
}

export function setAuxContent(content) {
    return authorizedRequest('/api/content/set/aux', { content, type: 'TEXT' });
}

export function saveMedia(payload) {
    return authorizedRequest('/api/media', payload);
}

export function moveMedia(body) {
    return authorizedRequest('/api/media/move', body, 'PUT');
}

export function sendCommand({ contentId, content }) {
    return commandRequest({ contentId, type: 'COMMAND', content });
}

export async function fetchSongs() {
    const res = await fetch('/api/songs');
    return res.json();
}

export async function fetchSongContent(song) {
    const params = new URLSearchParams({ song }).toString();
    const res = await fetch(`/api/songs/content?${params}`);
    return res.text();
}

export async function fetchSongFolders(archive) {
    const params = new URLSearchParams({ archive }).toString();
    const res = await fetch(`/api/songs/folders?${params}`);
    return res.json();
}

export async function fetchSongsFromFolder(archive, folder) {
    const params = new URLSearchParams({ archive, folder }).toString();
    const res = await fetch(`/api/songs/folder?${params}`);
    return res.json();
}

export async function fetchLyricsByUrl(url) {
    const params = new URLSearchParams({ url }).toString();
    const res = await fetch(`/api/lyrics/letras/song?${params}`);
    return res.text();
}

export async function fetchLyricsByArtistAndSong(artista, musica) {
    const params = new URLSearchParams({ artista, musica }).toString();
    const res = await fetch(`/api/lyrics/letras?${params}`);
    return res.text();
}
