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

const REMOVE_FROM_LIST_DESTINATION = 'musicas/_removidas';
const CATEGORY_SONGS = 'songs';

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
}

function removeBlockSelectedClass() {
    document.querySelectorAll('.bloco-selected').forEach(elemQuery => {
        const elem = document.getElementById(elemQuery.id).classList.remove('bloco-selected');
    });
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
    return { title, author, category, content }
}

function setAuxTextToBackend(content) {
    return setAuxContent(content);
}

async function saveProcessedTextToBackend(content) {
    const payload = getSongToSavePayload(content);
    await saveMedia(payload);
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
        destination: REMOVE_FROM_LIST_DESTINATION
    };
    await moveMedia(body);
    getSongList();
    getSongFolders();
    displayAlert('Texto removido da lista com sucesso');
}
// TODO: criar um elemento para cada alerta em vez de todos reaproveitarem
function displayAlert(conteudo) {
    const alertBox = document.getElementById('alert-modal');
    const alertContent = document.getElementById('alert-text');
    alertContent.textContent = conteudo;
    alertBox.style.display = 'block';
    alertBox.style.opacity = 100;
    setTimeout(t => {
        alertContent.textContent = '';
        alertBox.style.opacity = 0;
        alertBox.style.display = 'none';
    }, 4000);
}
document.addEventListener('DOMContentLoaded', () => {
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
        document.getElementById('folder-options').addEventListener('change', loadSongListFromFolder);
        document.getElementById('modal').classList.remove('hide');
        document.getElementById('buscar-musica').classList.remove('hide');
    });
    document.getElementById('close-modal-procurar').addEventListener('click', () => {
        document.getElementById('folder-options').removeEventListener('change', loadSongListFromFolder);
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
