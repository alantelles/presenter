import {
    emitMainContent,
    setAuxContent,
    saveMedia,
    moveMedia,
    sendCommand,
    fetchSongs,
    fetchSongContent,
    fetchSongFolders,
    fetchSongsFromFolder,
    fetchLyricsByUrl,
    fetchLyricsByArtistAndSong,
} from './client.js';
import { CHAT_NAME_STORAGE_KEY, initializeChat, sendSongLoadCommand } from './chat.js';

const REMOVE_FROM_LIST_DESTINATION = 'musicas/_removidas';
const CATEGORY_SONGS = 'songs';
let songCatalog = [];

function createElemPs(line) {
    return line.split('\n').map(line => {
        const pNode = document.createElement("p");
        const textNode = document.createTextNode(line + '\n');
        pNode.appendChild(textNode);
        return pNode;
    });
}

function createBlocks(content = null) {
    let textToShow;
    if (typeof(content) === 'string') {
        textToShow = content.trim();
    } else {
        textToShow = document.getElementById('text-to-show').value.trim();
        saveProcessedText(textToShow);
    }
    const splittedParagraphs = textToShow.split('\n\n')
        .map(line => line.trim())
        .filter(e => e.length > 0);
    let index = 0;
    const blocks = []; // splittedParagraphs.map(createBlock, index);
    for (let i = 0; i < splittedParagraphs.length; i++) {
        const blockResult = createBlock(splittedParagraphs[i], -1);
        if (!blockResult.isComment) {
            blockResult.block.id = `bl${index}`;
            index++;
        }
        blocks.push(blockResult.block);
    }
    const elemBlocos = document.getElementById('blocos');
    elemBlocos.textContent = '';
    blocks.forEach(block => {
        elemBlocos.appendChild(block);
    });
    addGoLiveEventListener();
}

function createBlock(paragraph, index) {
    const block = document.createElement('div');
    block.classList.add('bloco');
    const isComment = paragraph[0] === ':';
    if (isComment) {
        paragraph = paragraph.substr(1);
        block.classList.add('bloco-comment');
    } else {
        block.classList.add('bloco-verse');
    }
    const container = document.createElement('div');
    const ps = createElemPs(paragraph);
    ps.forEach(p => {
        container.appendChild(p);
    })
    block.appendChild(container);
    return {block, isComment};
}

async function sendBlockToLive() {
    const nextId = `bl${parseInt(this.id.replace('bl', '')) + 1}`;
    let nextBlock = '';
    try {
        nextBlock = document.getElementById(nextId).textContent
            .split('\n')
            .map(e => e.trim())
            .filter(e => !e.startsWith('meta:'))
            .slice(0, 3)
            .map(e => `meta:${e}`)
            .join('\n');

    } catch (e) {
        console.log(`Não foi possível extrair próximo bloco: ${e}`);
    }
    const content = this.textContent
        .split('\n')
        .map(e => e.trim())
        .filter(e => e.length > 1)
        .join('\n') + '\n' + nextBlock;
    const clicked = document.getElementById(this.id);
    clicked.classList.add('bloco-waiting');
    await emitContent(content);
    removeBlockSelectedClass();
    clicked.classList.remove('bloco-waiting');
    clicked.classList.add('bloco-selected');
    clicked.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

function removeBlockSelectedClass() {
    document.querySelectorAll('.bloco-selected').forEach(elemQuery => {
        const elem = document.getElementById(elemQuery.id).classList.remove('bloco-selected');
    });
}

function navigateSelectedBlock(event) {
    if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') {
        return;
    }
    if (event.target.matches('input, textarea, select, [contenteditable="true"]')) {
        return;
    }

    const blocks = Array.from(document.querySelectorAll('.bloco-verse'));
    const selectedBlock = document.querySelector('.bloco-verse.bloco-selected');
    const selectedIndex = blocks.indexOf(selectedBlock);
    const nextIndex = selectedIndex + (event.key === 'ArrowUp' ? -1 : 1);
    if (selectedIndex < 0 || nextIndex < 0 || nextIndex >= blocks.length) {
        return;
    }

    event.preventDefault();
    blocks[nextIndex].click();
}

function triggerTimer(horas, minutos) {
    const now = new Date();
    const horaAlvo = new Date(now.getFullYear(), now.getMonth(), now.getDate(), horas, minutos, 0);
    const diff = Math.round((horaAlvo.getTime() - now.getTime()) / 1000); // converte para segundos
    if (diff < 0) {
        return;
    }
    sendCommand({
        contentId: `cronometro-${Date.now()}`,
        content: `regressivo ${diff}`
    });
}

function clearTimer() {
    sendCommand({
        contentId: `limparCronometro-${Date.now()}`
    });
}

function emitContent(content) {
    return emitMainContent(content);
}

function addGoLiveEventListener() {
    const blocks = document.querySelectorAll('.bloco-verse');
    blocks.forEach(element => {
        element.addEventListener('click', sendBlockToLive);
    });
}

function getSongToSavePayload(content) {
    // assumindo padrão cifra club
    const titleParts = content.split("\n\n")[0]
        .split("\n")
    const title = titleParts[0]
    const author = titleParts.length == 2 ? titleParts[1] : "Artista desconhecido"
    const category = "songs"
    const username = window.localStorage.getItem(CHAT_NAME_STORAGE_KEY) || '';
    return { title, author, category, content, username }
}

function setAuxTextToBackend(content) {
    return setAuxContent(content);
}

async function saveProcessedTextToBackend(content) {
    const payload = getSongToSavePayload(content);
    const response = await saveMedia(payload);
    if (!response.ok) {
        displayAlert('Não foi possível salvar o texto.');
        return;
    }
    getSongList();
    displayAlert('Texto salvo com sucesso!');
}

function saveProcessedText(content) {
    window.localStorage.setItem('lastText', content);
}

function openProcessedText() {
    const lastText = window.localStorage.getItem('lastText');
    if (lastText) {
        document.getElementById('text-to-show').value = lastText;
    }
}

function clearTextInput() {
    const textToShow = document.getElementById('text-to-show');
    textToShow.value = '';
    return textToShow;
}

async function pasteToTextInput() {
    const elem = document.getElementById('text-to-show');
    const text = await navigator.clipboard.readText();
    elem.value = text;
}

async function fillTextInput() {
    const fileName = document.getElementById('select-text-options').value;
    clearTextInput().value = await loadTextFromBackend(fileName);
}
function loadTextForAll() {
    const fileName = document.getElementById('select-text-options').value;
    if (!fileName) {
        displayAlert('Selecione um texto para carregar.');
        return;
    }
    if (!sendSongLoadCommand(fileName)) {
        displayAlert('Chat desconectado. Não foi possível carregar para todos.');
    }
}
async function loadNamedTextIntoController(event) {
    const songName = event.detail.songName;
    try {
        document.getElementById('text-to-show').value = await loadTextFromBackend(songName);
        displayAlert(`${event.detail.username} carregou ${songName}`);
    } catch (error) {
        displayAlert(`Não foi possível carregar ${songName}.`);
    }
}
async function loadAndCreateBlocks() {
    const fileName = document.getElementById('select-text-options').value;
    const text = await loadTextFromBackend(fileName);
    createBlocks(text);
    toggleControllerForce();
}
async function loadAndCreateBlocksFromTrigger(event) {
    const fileName = event.target.value;
    const text = await loadTextFromBackend(fileName);
    createBlocks(text);
    toggleControllerForce();
}

async function loadTextFromBackend(song) {
    const cached = window.sessionStorage.getItem(`text:${song}`);
    if (cached) {
        return cached;
    }
    const content = fetchSongContent(song);
    cacheLoadedText(song, content);
    return content
}

async function cacheLoadedText(song, content$) {
    const content = await content$
    window.sessionStorage.setItem(`text:${song}`, content);
}

async function getSongList() {
    const json = await fetchSongs();
    fillSongsOption(json.mediaList);
    fillSongsInButtonTriggers(json.mediaList);
    clearAllTextCache();
}

async function triggerGetSongList() {
    await getSongList();
    displayAlert('Lista atualizada com sucesso');
}

function clearAllTextCache() {
    Object.keys(window.sessionStorage).forEach((key) => {
        if (key.startsWith('text:')) {
            window.sessionStorage.removeItem(key)
        }
    })
}

async function getSongFolders() {
    const json = await fetchSongFolders('musicas');
    fillSongsFoldersOption(json.mediaList);
}

async function getLyricsFromLetrasByUrl() {
    const url = document.getElementById('url-busca-musica').value;
    const lyrics = await fetchLyricsByUrl(url);
    document.getElementById('text-to-show').value = lyrics
    document.getElementById('close-modal-letras').click();
}

async function getLyricsFromLetrasByAndArtist() {
    const artista = document.getElementById('artista-busca-musica').value;
    const musica = document.getElementById('nome-busca-musica').value;
    const lyrics = await fetchLyricsByArtistAndSong(artista, musica);
    if (lyrics.length == 0) {
        displayAlert('Letra não encontrada');
        return;
    }
    document.getElementById('text-to-show').value = lyrics
    document.getElementById('modal').classList.add('hide');
}

async function loadSongListFromFolder() {
    const folder = document.getElementById('folder-options').value;
    const archive = 'musicas'
    const json = await fetchSongsFromFolder(archive, folder);
    fillSongListFromFolderTable(json.mediaList, folder, archive);
}

async function loadSongCatalog() {
    const folders = (await fetchSongFolders('musicas')).mediaList
        .filter(folder => !folder.startsWith('_'));
    const songsByFolder = await Promise.all(folders.map(async folder => {
        const json = await fetchSongsFromFolder('musicas', folder);
        return json.mediaList.map(song => ({
            name: song,
            path: `musicas/${folder}/${song}`
        }));
    }));
    songCatalog = songsByFolder.flat().sort((first, second) =>
        first.name.localeCompare(second.name, 'pt-BR', { sensitivity: 'base' })
    );
}

function normalizeSongSearch(value) {
    return value.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLocaleLowerCase('pt-BR');
}

function searchSongs() {
    const query = normalizeSongSearch(document.getElementById('buscar-musicas').value.trim());
    if (!query) {
        loadSongListFromFolder();
        return;
    }
    const matches = songCatalog.filter(song =>
        normalizeSongSearch(song.name.replace(/\.txt$/i, '')).includes(query)
    );
    fillSongSearchResults(matches);
}

function fillSongSearchResults(songs) {
    const tbody = document.getElementById('tabela-lista-musicas');
    tbody.textContent = '';
    if (songs.length === 0) {
        const row = document.createElement('tr');
        const cell = document.createElement('td');
        cell.colSpan = 2;
        cell.textContent = 'Nenhuma música encontrada';
        row.appendChild(cell);
        tbody.appendChild(row);
        return;
    }
    songs.forEach(song => {
        const row = document.createElement('tr');
        const buttonCell = document.createElement('td');
        buttonCell.classList.add('tg0-lax', 'autosize');
        buttonCell.appendChild(createButtonLoadSongNode(song.path));
        const nameCell = document.createElement('td');
        nameCell.classList.add('tg0-lax');
        nameCell.textContent = song.name;
        row.append(buttonCell, nameCell);
        tbody.appendChild(row);
    });
    addLoadSongFromFolderEventListener();
}

function fillSongListFromFolderTable(songs, folder, archive) {
    const tbody = document.getElementById('tabela-lista-musicas');
    tbody.textContent = '';
    songs.forEach(song => {
        const trNode = document.createElement('tr');
        const tdButtonNode = document.createElement('td');
        tdButtonNode.classList.add('tg0-lax', 'autosize');
        const path = `${archive}/${folder}/${song}`;
        tdButtonNode.appendChild(createButtonLoadSongNode(path));
        const tdSongNameNode = document.createElement('td');
        tdSongNameNode.classList.add('tg0-lax');
        tdSongNameNode.textContent = song;
        trNode.append(tdButtonNode, tdSongNameNode);
        tbody.appendChild(trNode);
    });
    addLoadSongFromFolderEventListener();
}

function addLoadSongFromFolderEventListener() {
    const loaders = document.querySelectorAll('.song-loader');
    loaders.forEach(elem => {
        elem.addEventListener('click', async function() {
            const song = this.value;
            clearTextInput().value = await loadTextFromBackend(song);
            document.getElementById('close-modal-procurar').click();
        });
    })
}

function createButtonLoadSongNode(path) {
    const btnNode = document.createElement('button');
    btnNode.classList.add('button', 'button-small', 'song-loader');
    btnNode.value = path;
    btnNode.textContent = 'Carregar';
    return btnNode;
}

function fillSongsOption(mediaList) {
    const selectSong = document.getElementById('select-text-options');
    selectSong.textContent = '';
    mediaList.sort().forEach(songName => {
        const optionElem = document.createElement('option');
        optionElem.value = songName;
        optionElem.textContent = songName;
        selectSong.appendChild(optionElem);
    });
}

function fillSongsInButtonTriggers(mediaList) {
    const textTriggersContainer = document.getElementById('text-triggers');
    textTriggersContainer.textContent = '';
    mediaList.sort().forEach(songName => {
        const triggerElem = document.createElement('button');
        triggerElem.classList.add('button', 'button-big', 'text-trigger-button')
        triggerElem.value = songName;
        triggerElem.textContent = songName.replace('.txt', '');
        triggerElem.addEventListener('click', loadAndCreateBlocksFromTrigger);
        textTriggersContainer.appendChild(triggerElem);
    });
}

function fillSongsFoldersOption(mediaList) {
    const selectSong = document.getElementById('folder-options');
    selectSong.textContent = '';
    mediaList.sort().forEach(folder => {
        const optionElem = document.createElement('option');
        optionElem.value = folder;
        optionElem.textContent = folder;
        selectSong.appendChild(optionElem);
    });
}
function toggleVideo() {
    const toggleVideoCheckbox = document.getElementById('toggle-video');
    sendCommand({
        contentId: `toggleVideo-${Date.now()}`,
        content: toggleVideoCheckbox.checked ? 'on' : 'off'
    });
}
function toggleController(event) {
    _toggleController(event);
}
function toggleControllerForce() {
    _toggleController(null);
}
function _toggleController(event) {
    const controller = document.getElementById('controller-panel');
    const blocks = document.getElementById('blocks-panel');
    const toggler = document.getElementById('toggle-controller-button');
    const togglerContainer = document.getElementById('toggle-controller-container');
    if (!togglerContainer.checkVisibility()) {
        return;
    }
    const showed = controller.style.display !== 'none';
    if (showed || !event) { // esconder painel
        blocks.style.width = '100%';
        blocks.style.left = '0';
        controller.style.display = 'none';
        blocks.style.display = 'block';
        toggler.classList.add('toggle-off');
        toggler.innerHTML = '&#x2699;'
    } else {
        controller.style.width = '100%';
        blocks.style.display = 'none';
        controller.style.display = 'block';
        toggler.classList.remove('toggle-off');
        toggler.innerHTML = '&#x274F;'
    }
}
async function removeFromList() {
    const mediaId = document.getElementById('select-text-options').value;
    const body = {
        mediaId,
        category: CATEGORY_SONGS,
        destination: REMOVE_FROM_LIST_DESTINATION,
        username: window.localStorage.getItem(CHAT_NAME_STORAGE_KEY) || ''
    };
    const response = await moveMedia(body);
    if (!response.ok) {
        displayAlert('Não foi possível remover o texto da lista.');
        return;
    }
    getSongList();
    getSongFolders();
    displayAlert('Texto removido da lista com sucesso');
}
let alertTimeout;
function displayAlert(conteudo) {
    const alertBox = document.getElementById('alert-modal');
    const alertContent = document.getElementById('alert-text');
    alertContent.textContent = conteudo;
    alertBox.style.display = 'block';
    alertBox.style.opacity = 100;
    window.clearTimeout(alertTimeout);
    alertTimeout = window.setTimeout(() => {
        alertContent.textContent = '';
        alertBox.style.opacity = 0;
        alertBox.style.display = 'none';
    }, 4000);
}
document.addEventListener('DOMContentLoaded', () => {
    initializeChat();
    document.addEventListener('keydown', navigateSelectedBlock);
    window.addEventListener('presenter:refresh-songs', () => getSongList());
    window.addEventListener('presenter:load-song-by-name', loadNamedTextIntoController);
    window.addEventListener('presenter:new-chat-message', event => {
        displayAlert(`Nova mensagem de ${event.detail.username}`);
    });
    addGoLiveEventListener();
    document.getElementById('process-text').addEventListener('click', (event) => {
        createBlocks(event);
        toggleControllerForce();
    });
    document.getElementById('clear-screen').addEventListener('click', () => {
        removeBlockSelectedClass();
        emitContent('');
    });
    document.getElementById('toggle-controller-button').addEventListener('click', toggleController);
    document.getElementById('carregar-lista-musicas').addEventListener('click', triggerGetSongList);
    document.getElementById('clear-text-input').addEventListener('click', clearTextInput);
    document.getElementById('get-lyrics-letras-by-url').addEventListener('click', getLyricsFromLetrasByUrl);
    document.getElementById('get-lyrics-letras-by-artista-musica').addEventListener('click', getLyricsFromLetrasByAndArtist);
    // document.getElementById('paste-to-text-input').addEventListener('click', pasteToTextInput);
    document.getElementById('save-text').addEventListener('click', () => {
        const content = document.getElementById('text-to-show').value.trim();
        saveProcessedTextToBackend(content);
    });
    document.getElementById('toggle-controller-button-clear').addEventListener('click', () => {
        removeBlockSelectedClass();
        emitContent('');
    });
    document.getElementById('set-aux-text').addEventListener('click', () => {
        const content = document.getElementById('text-to-show').value;
        setAuxTextToBackend(content);
    });
    document.getElementById('clean-aux-text').addEventListener('click', () => {
        setAuxTextToBackend('');
    });
    document.getElementById('carregar-texto').addEventListener('click', fillTextInput);
    document.getElementById('carregar-texto-para-todos').addEventListener('click', loadTextForAll);
    document.getElementById('carregar-blocos').addEventListener('click', loadAndCreateBlocks);
    document.getElementById('remover-texto').addEventListener('click', removeFromList);
    // document.getElementById('open-modal-letras').addEventListener('click', () => {
    //     document.getElementById('modal').classList.remove('hide');
    //     document.getElementById('buscar-letra').classList.remove('hide');
    // });
    // document.getElementById('close-modal-letras').addEventListener('click', () => {
    //     document.getElementById('modal').classList.add('hide');
    //     document.getElementById('buscar-letra').classList.add('hide');
    // });
    document.getElementById('open-modal-procurar').addEventListener('click', async () => {
        await loadSongListFromFolder();
        await loadSongCatalog();
        document.getElementById('buscar-musicas').value = '';
        document.getElementById('folder-options').addEventListener('change', loadSongListFromFolder);
        document.getElementById('buscar-musicas').addEventListener('input', searchSongs);
        document.getElementById('modal').classList.remove('hide');
        document.getElementById('buscar-musica').classList.remove('hide');
    });
    document.getElementById('close-modal-procurar').addEventListener('click', () => {
        document.getElementById('folder-options').removeEventListener('change', loadSongListFromFolder);
        document.getElementById('buscar-musicas').removeEventListener('input', searchSongs);
        document.getElementById('modal').classList.add('hide');
        document.getElementById('buscar-musica').classList.add('hide');
    });
    document.getElementById('carregar-lista-pasta').addEventListener('click', loadSongListFromFolder);

    document.getElementById('trigger-timer-1930').addEventListener('click', triggerTimer.bind(null, 19, 30));
    document.getElementById('trigger-timer-1800').addEventListener('click', triggerTimer.bind(null, 18, 0));
    document.getElementById('clear-regressive').addEventListener('click', clearTimer);

    document.getElementById('toggle-video').addEventListener('change', toggleVideo);

    openProcessedText();
    createBlocks();
    getSongList();
    getSongFolders();
    setInnerMarginTop();
});
function setInnerMarginTop() {
    const fixed = document.querySelector('.fixed');
    const inner = document.querySelector('.inner');
    // const fixedHeight = fixed.offsetHeight;
    // console.log(`Fixed height: ${fixedHeight}`);
    // inner.style.marginTop = `${fixedHeight}px`;
}
