import {
    uploadImage,
    fetchImageList,
    uploadVideo,
    fetchVideoList,
    deleteVideo,
    emitImageContent,
    emitVideoContent,
    clearMainContent,
    fetchCurrentContent,
    playVideo,
    pauseVideo,
    forwardVideo,
    rewindVideo,
    restartVideo,
    thumbUrl,
    videoContentUrl,
    fetchDownloaderStatus,
    downloadVideoWithProgress,
    withBasePath
} from './client.js';

const state = {
    images: [],
    videos: [],
    activeFilter: 'all',
    searchQuery: '',
    currentMain: null,
    selectedItem: null, // { name: string, type: 'IMAGE' | 'VIDEO' }
};

let toastTimeout = null;
function showToast(message, duration = 3000) {
    const toast = document.getElementById('toast');
    if (!toast) return;
    toast.textContent = message;
    toast.style.display = 'block';
    clearTimeout(toastTimeout);
    toastTimeout = setTimeout(() => {
        toast.style.display = 'none';
    }, duration);
}

async function loadCurrentContent() {
    try {
        state.currentMain = await fetchCurrentContent();
        updateVideoControlBar();
        updateActiveCardHighlight();
    } catch (e) {
        console.warn('Erro ao carregar conteúdo atual:', e);
    }
}

const playback = {
    videoName: null,
    isPlaying: false,
    currentTime: 0,
    duration: 0,
    lastTick: null,
    intervalId: null,
};

function formatVideoTime(seconds) {
    if (isNaN(seconds) || seconds < 0) return '00:00';
    const s = Math.floor(seconds);
    const m = Math.floor(s / 60);
    const sec = s % 60;
    const h = Math.floor(m / 60);
    const min = m % 60;
    if (h > 0) {
        return `${h}:${min.toString().padStart(2, '0')}:${sec.toString().padStart(2, '0')}`;
    }
    return `${min.toString().padStart(2, '0')}:${sec.toString().padStart(2, '0')}`;
}

function startPlaybackTicker() {
    if (playback.intervalId) return;
    playback.lastTick = performance.now();
    playback.intervalId = setInterval(() => {
        const now = performance.now();
        const delta = (now - playback.lastTick) / 1000;
        playback.lastTick = now;

        if (playback.isPlaying && playback.duration > 0) {
            playback.currentTime += delta;
            if (playback.currentTime >= playback.duration) {
                playback.currentTime = playback.duration;
                playback.isPlaying = false;
            }
            updateTimerDisplay();
        }
    }, 250);
}

function stopPlaybackTicker() {
    if (playback.intervalId) {
        clearInterval(playback.intervalId);
        playback.intervalId = null;
    }
}

function updateTimerDisplay() {
    const cur = playback.currentTime || 0;
    const dur = playback.duration || 0;
    const rem = Math.max(0, dur - cur);

    const curEl = document.getElementById('ctrl-video-current');
    const durEl = document.getElementById('ctrl-video-duration');
    const remEl = document.getElementById('ctrl-video-remaining');
    const progEl = document.getElementById('ctrl-video-progress');
    const badgeEl = document.getElementById('video-status-badge');

    if (curEl) curEl.textContent = formatVideoTime(cur);
    if (durEl) durEl.textContent = formatVideoTime(dur);
    if (remEl) remEl.textContent = formatVideoTime(rem);

    if (progEl) {
        const pct = dur > 0 ? (cur / dur) * 100 : 0;
        progEl.style.width = `${pct}%`;
    }

    const isVideoLive = state.currentMain && state.currentMain.type === 'VIDEO' && Boolean(state.currentMain.content);
    if (badgeEl && isVideoLive) {
        if (playback.isPlaying) {
            badgeEl.className = 'status-badge live';
            badgeEl.textContent = 'NO AR';
        } else {
            badgeEl.className = 'status-badge paused';
            badgeEl.textContent = 'PAUSADO';
        }
    }
}

function fetchVideoDuration(videoName) {
    return new Promise((resolve) => {
        const cardVid = document.querySelector(`.media-card[data-name="${CSS.escape(videoName)}"][data-type="VIDEO"] video`);
        if (cardVid && cardVid.duration && !isNaN(cardVid.duration) && cardVid.duration > 0) {
            resolve(cardVid.duration);
            return;
        }

        const probe = document.createElement('video');
        probe.preload = 'metadata';
        probe.src = videoContentUrl(videoName);
        probe.onloadedmetadata = () => {
            resolve(probe.duration || 0);
        };
        probe.onerror = () => resolve(0);
    });
}

async function updateVideoControlBar() {
    const panel = document.getElementById('video-control-panel');
    const badge = document.getElementById('video-status-badge');
    const display = document.getElementById('current-video-display');
    const timerRow = document.getElementById('video-timer-row');
    const isVideoLive = state.currentMain && state.currentMain.type === 'VIDEO' && Boolean(state.currentMain.content);

    // Desabilita os botões de controle se nenhum vídeo estiver rodando
    const videoButtons = panel.querySelectorAll('.video-controls-row .btn');
    videoButtons.forEach(btn => {
        btn.disabled = !isVideoLive;
    });

    if (isVideoLive) {
        panel.classList.add('active');
        display.innerHTML = `<span>Em reprodução no telão:</span> <span class="current-video-name">${escapeHtml(state.currentMain.content)}</span>`;
        if (timerRow) timerRow.style.display = 'block';

        if (playback.videoName !== state.currentMain.content) {
            playback.videoName = state.currentMain.content;
            playback.currentTime = 0;
            playback.isPlaying = false; // Ao carregar ou atualizar tela (F5), não roda automaticamente
            playback.duration = await fetchVideoDuration(state.currentMain.content);
            updateTimerDisplay();
        } else {
            updateTimerDisplay();
        }
    } else {
        panel.classList.remove('active');
        badge.className = 'status-badge idle';
        badge.textContent = 'Parado';
        if (timerRow) timerRow.style.display = 'none';

        playback.videoName = null;
        playback.isPlaying = false;
        playback.currentTime = 0;
        playback.duration = 0;
        stopPlaybackTicker();

        if (state.currentMain && state.currentMain.type === 'IMAGE' && state.currentMain.content) {
            display.innerHTML = `<span>Imagem no ar:</span> <span class="current-video-name">${escapeHtml(state.currentMain.content)}</span>`;
        } else {
            display.innerHTML = '<span>Nenhum conteúdo em exibição no momento.</span>';
        }
    }
}

function selectItem(item) {
    if (state.selectedItem && state.selectedItem.name === item.name && state.selectedItem.type === item.type) {
        // Se clicar no já selecionado, mantém selecionado
        return;
    }
    state.selectedItem = item;
    updateSelectionBar();
    updateActiveCardHighlight();
}

function clearSelection() {
    state.selectedItem = null;
    updateSelectionBar();
    updateActiveCardHighlight();
}

function updateSelectionBar() {
    const panel = document.getElementById('selection-panel');
    const badge = document.getElementById('selection-badge');
    const display = document.getElementById('selection-display');
    const goLiveBtn = document.getElementById('btn-go-live');
    const clearBtn = document.getElementById('btn-clear-selection');

    if (state.selectedItem) {
        panel.classList.add('has-selection');
        const isVideo = state.selectedItem.type === 'VIDEO';
        badge.className = isVideo ? 'status-badge badge-video' : 'status-badge badge-image';
        badge.textContent = isVideo ? 'Vídeo Selecionado' : 'Imagem Selecionada';
        display.innerHTML = `<span>Pronto para exibir:</span> <span class="selected-name">${escapeHtml(state.selectedItem.name)}</span>`;
        
        goLiveBtn.disabled = false;
        goLiveBtn.className = 'btn btn-primary';
        goLiveBtn.innerHTML = isVideo ? '▶ Dar o Play no Vídeo' : '🖼️ Exibir Imagem no Telão';
        clearBtn.style.display = 'inline-flex';
    } else {
        panel.classList.remove('has-selection');
        badge.className = 'status-badge idle';
        badge.textContent = 'Nenhuma';
        display.innerHTML = '<span class="selection-hint">Clique em um card abaixo para selecionar uma imagem ou vídeo.</span>';
        goLiveBtn.disabled = true;
        goLiveBtn.innerHTML = '▶ Exibir no Telão';
        clearBtn.style.display = 'none';
    }
}

function updateActiveCardHighlight() {
    const cards = document.querySelectorAll('.media-card');
    cards.forEach(card => {
        const type = card.dataset.type;
        const name = card.dataset.name;
        const isLive = state.currentMain && state.currentMain.type === type && state.currentMain.content === name;
        const isSelected = state.selectedItem && state.selectedItem.type === type && state.selectedItem.name === name;

        // Destaque de No Ar (Live)
        if (isLive) {
            card.classList.add('is-active');
            let liveTag = card.querySelector('.live-tag');
            if (!liveTag) {
                liveTag = document.createElement('div');
                liveTag.className = 'live-tag';
                liveTag.textContent = '● NO AR';
                card.querySelector('.media-thumb-container').appendChild(liveTag);
            }
        } else {
            card.classList.remove('is-active');
            const liveTag = card.querySelector('.live-tag');
            if (liveTag) {
                liveTag.remove();
            }
        }

        // Destaque de Selecionado
        if (isSelected) {
            card.classList.add('is-selected');
            let selTag = card.querySelector('.selected-tag');
            if (!selTag) {
                selTag = document.createElement('div');
                selTag.className = 'selected-tag';
                selTag.textContent = '✓ SELECIONADO';
                card.querySelector('.media-thumb-container').appendChild(selTag);
            }
        } else {
            card.classList.remove('is-selected');
            const selTag = card.querySelector('.selected-tag');
            if (selTag) {
                selTag.remove();
            }
        }

        // Atualiza texto e classe do botão de ação do card
        const cardActionBtn = card.querySelector('.btn-card-action');
        if (cardActionBtn) {
            const isVideo = type === 'VIDEO';
            if (isSelected) {
                cardActionBtn.className = 'btn btn-card-action btn-success';
                cardActionBtn.innerHTML = isVideo ? '▶ Dar o Play' : '🖼️ Exibir no Telão';
            } else {
                cardActionBtn.className = 'btn btn-card-action btn-secondary';
                cardActionBtn.innerHTML = 'Selecionar';
            }
        }
    });
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

async function loadMediaData() {
    try {
        const [imgs, vids] = await Promise.all([
            fetchImageList().catch(err => {
                console.error('Erro ao listar imagens:', err);
                return [];
            }),
            fetchVideoList().catch(err => {
                console.error('Erro ao listar vídeos:', err);
                return [];
            })
        ]);
        state.images = imgs;
        state.videos = vids;
        updateTabCounters();
        renderGallery();
    } catch (err) {
        showToast('Erro ao carregar lista de mídias: ' + err.message);
    }
}

function updateTabCounters() {
    const tabAll = document.getElementById('tab-all');
    const tabImages = document.getElementById('tab-images');
    const tabVideos = document.getElementById('tab-videos');

    tabAll.textContent = `Todas (${state.images.length + state.videos.length})`;
    tabImages.textContent = `Imagens (${state.images.length})`;
    tabVideos.textContent = `Vídeos (${state.videos.length})`;
}

function renderGallery() {
    const gallery = document.getElementById('gallery');
    gallery.innerHTML = '';

    const query = state.searchQuery.toLowerCase().trim();
    const mediaItems = [];

    if (state.activeFilter === 'all' || state.activeFilter === 'images') {
        state.images.forEach(name => {
            if (!query || name.toLowerCase().includes(query)) {
                mediaItems.push({ name, type: 'IMAGE' });
            }
        });
    }

    if (state.activeFilter === 'all' || state.activeFilter === 'videos') {
        state.videos.forEach(name => {
            if (!query || name.toLowerCase().includes(query)) {
                mediaItems.push({ name, type: 'VIDEO' });
            }
        });
    }

    // Ordenação alfabética insensível a maiúsculas/minúsculas e natural com números
    mediaItems.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base', numeric: true }));

    if (mediaItems.length === 0) {
        const empty = document.createElement('div');
        empty.className = 'empty-gallery';
        empty.textContent = query
            ? 'Nenhuma mídia encontrada para a busca.'
            : 'Nenhuma mídia encontrada. Faça upload de imagens ou vídeos acima.';
        gallery.appendChild(empty);
        return;
    }

    for (const item of mediaItems) {
        const isVideo = item.type === 'VIDEO';
        const isSelected = state.selectedItem && state.selectedItem.type === item.type && state.selectedItem.name === item.name;

        const card = document.createElement('div');
        card.className = `media-card ${isSelected ? 'is-selected' : ''}`;
        card.dataset.type = item.type;
        card.dataset.name = item.name;

        const thumbContainer = document.createElement('div');
        thumbContainer.className = 'media-thumb-container';

        const badge = document.createElement('div');
        badge.className = `media-badge ${isVideo ? 'badge-video' : 'badge-image'}`;
        badge.textContent = isVideo ? 'VÍDEO' : 'IMAGEM';
        thumbContainer.appendChild(badge);

        if (isVideo) {
            const vid = document.createElement('video');
            vid.src = `${videoContentUrl(item.name)}#t=0.1`;
            vid.preload = 'metadata';
            vid.muted = true;
            vid.playsInline = true;
            thumbContainer.appendChild(vid);

            const durationBadge = document.createElement('div');
            durationBadge.className = 'video-duration-badge';
            durationBadge.textContent = '--:--';
            thumbContainer.appendChild(durationBadge);

            vid.addEventListener('loadedmetadata', () => {
                if (vid.duration && !isNaN(vid.duration)) {
                    durationBadge.textContent = formatVideoTime(vid.duration);
                }
            });
        } else {
            const img = document.createElement('img');
            img.src = thumbUrl(item.name);
            img.alt = item.name;
            img.loading = 'lazy';
            img.onerror = () => {
                img.style.display = 'none';
                thumbContainer.textContent = '🖼️ Imagem';
            };
            thumbContainer.appendChild(img);
        }

        const info = document.createElement('div');
        info.className = 'media-info';

        const title = document.createElement('div');
        title.className = 'media-name';
        title.textContent = item.name;
        title.title = item.name;
        info.appendChild(title);

        const actions = document.createElement('div');
        actions.className = 'media-actions';

        const actionBtn = document.createElement('button');
        actionBtn.className = `btn btn-card-action ${isSelected ? 'btn-success' : 'btn-secondary'}`;
        actionBtn.textContent = isSelected ? (isVideo ? '▶ Dar o Play' : '🖼️ Exibir') : 'Selecionar';
        actionBtn.addEventListener('click', async (e) => {
            e.stopPropagation();
            if (state.selectedItem && state.selectedItem.name === item.name && state.selectedItem.type === item.type) {
                await triggerGoLive(item);
            } else {
                selectItem(item);
            }
        });
        actions.appendChild(actionBtn);

        if (isVideo) {
            const delBtn = document.createElement('button');
            delBtn.className = 'btn-card-delete';
            delBtn.title = 'Excluir vídeo';
            delBtn.innerHTML = '🗑️';
            delBtn.addEventListener('click', async (e) => {
                e.stopPropagation();
                if (confirm(`Deseja realmente excluir o vídeo "${item.name}"?`)) {
                    try {
                        await deleteVideo(item.name);
                        showToast(`Vídeo "${item.name}" excluído.`);
                        if (state.selectedItem && state.selectedItem.name === item.name) {
                            clearSelection();
                        }
                        await loadMediaData();
                        await loadCurrentContent();
                    } catch (err) {
                        showToast('Erro ao excluir: ' + err.message);
                    }
                }
            });
            actions.appendChild(delBtn);
        }

        info.appendChild(actions);

        card.appendChild(thumbContainer);
        card.appendChild(info);

        // Ao clicar no card, apenas SELECIONA a mídia (não coloca no ar direto)
        card.addEventListener('click', () => {
            selectItem(item);
        });

        gallery.appendChild(card);
    }

    updateActiveCardHighlight();
}

async function triggerGoLive(item) {
    if (!item) return;
    try {
        if (item.type === 'VIDEO') {
            playback.videoName = item.name;
            playback.currentTime = 0;
            playback.isPlaying = true;
            startPlaybackTicker();
            playback.duration = await fetchVideoDuration(item.name);
            updateTimerDisplay();
            await emitVideoContent(item.name);
            showToast(`🎬 Vídeo "${item.name}" no ar!`);
        } else {
            playback.videoName = null;
            playback.isPlaying = false;
            playback.currentTime = 0;
            stopPlaybackTicker();
            await emitImageContent(item.name);
            showToast(`🖼 Imagem "${item.name}" no ar!`);
        }
        await loadCurrentContent();
    } catch (err) {
        showToast('Erro ao exibir: ' + err.message);
    }
}

function isVideoFile(file) {
    if (file.type && file.type.startsWith('video/')) return true;
    const ext = file.name.split('.').pop().toLowerCase();
    return ['mp4', 'webm', 'ogg', 'ogv', 'mov', 'm4v', 'mkv'].includes(ext);
}

async function handleFilesUpload(files) {
    if (!files || !files.length) return;

    const dropText = document.getElementById('drop-zone-text');
    const originalText = dropText.textContent;
    dropText.textContent = `Enviando ${files.length} arquivo(s)...`;

    let successCount = 0;
    let errors = [];

    for (const file of Array.from(files)) {
        try {
            if (isVideoFile(file)) {
                await uploadVideo(file);
            } else {
                await uploadImage(file);
            }
            successCount++;
        } catch (err) {
            errors.push(`${file.name}: ${err.message}`);
        }
    }

    dropText.textContent = originalText;

    if (successCount > 0) {
        showToast(`${successCount} arquivo(s) enviado(s) com sucesso!`);
    }
    if (errors.length > 0) {
        alert('Avisos durante o envio:\n' + errors.join('\n'));
    }

    await loadMediaData();
}

function setupUploadAndDropZone() {
    const dropZone = document.getElementById('drop-zone');
    const fileInput = document.getElementById('file-input');

    dropZone.addEventListener('click', () => fileInput.click());

    fileInput.addEventListener('change', async () => {
        await handleFilesUpload(fileInput.files);
        fileInput.value = '';
    });

    ['dragenter', 'dragover'].forEach(eventName => {
        dropZone.addEventListener(eventName, (e) => {
            e.preventDefault();
            e.stopPropagation();
            dropZone.classList.add('dragover');
        });
    });

    ['dragleave', 'drop'].forEach(eventName => {
        dropZone.addEventListener(eventName, (e) => {
            e.preventDefault();
            e.stopPropagation();
            dropZone.classList.remove('dragover');
        });
    });

    dropZone.addEventListener('drop', async (e) => {
        const dt = e.dataTransfer;
        const files = dt.files;
        await handleFilesUpload(files);
    });
}

function setupControlButtons() {
    // Botão de colocar a mídia selecionada no ar
    document.getElementById('btn-go-live').addEventListener('click', async () => {
        if (state.selectedItem) {
            await triggerGoLive(state.selectedItem);
        }
    });

    document.getElementById('btn-clear-selection').addEventListener('click', () => {
        clearSelection();
    });

    // Botões de controle de reprodução de vídeo
    document.getElementById('btn-video-restart').addEventListener('click', async () => {
        playback.currentTime = 0;
        playback.isPlaying = false; // Entra em pausa no início
        updateTimerDisplay();
        await restartVideo();
        showToast('⏮ Vídeo reiniciado no início (0:00) e pausado.');
    });

    document.getElementById('btn-video-rewind').addEventListener('click', async () => {
        // Se estava pausado, CONTINUA PAUSADO! O timer não roda sozinho!
        playback.currentTime = Math.max(0, playback.currentTime - 10);
        playback.lastTick = performance.now();
        updateTimerDisplay();
        await rewindVideo(10);
        showToast('⏪ Comando enviado: Voltar -10s');
    });

    document.getElementById('btn-video-play').addEventListener('click', async () => {
        playback.isPlaying = true;
        if (playback.duration > 0 && playback.currentTime >= playback.duration) {
            playback.currentTime = 0;
        }
        playback.lastTick = performance.now();
        startPlaybackTicker();
        updateTimerDisplay();
        await playVideo();
        showToast('▶ Comando enviado: Reproduzir vídeo');
    });

    document.getElementById('btn-video-pause').addEventListener('click', async () => {
        playback.isPlaying = false;
        updateTimerDisplay();
        await pauseVideo();
        showToast('⏸ Comando enviado: Pausar vídeo');
    });

    document.getElementById('btn-video-forward').addEventListener('click', async () => {
        playback.currentTime = Math.min(playback.duration || 0, playback.currentTime + 10);
        playback.lastTick = performance.now();
        updateTimerDisplay();
        await forwardVideo(10);
        showToast('⏩ Comando enviado: Avançar +10s');
    });

    const stopPlayback = async () => {
        playback.isPlaying = false;
        playback.currentTime = 0;
        playback.videoName = null;
        playback.duration = 0;
        stopPlaybackTicker();
        await clearMainContent();
        showToast('⏹ Telão limpo.');
        await loadCurrentContent();
    };

    document.getElementById('btn-video-stop').addEventListener('click', stopPlayback);
    document.getElementById('clear-main').addEventListener('click', stopPlayback);
}

function setupTabsAndSearch() {
    const tabs = document.querySelectorAll('.tab-btn');
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            state.activeFilter = tab.dataset.filter;
            renderGallery();
        });
    });

    const searchInput = document.getElementById('search-input');
    searchInput.addEventListener('input', (e) => {
        state.searchQuery = e.target.value;
        renderGallery();
    });
}

let downloaderAvailable = false;

async function checkDownloaderToolsStatus() {
    const alertBox = document.getElementById('tools-missing-alert');
    const alertMsg = document.getElementById('tools-missing-message');
    const submitBtn = document.getElementById('btn-submit-download');

    try {
        const status = await fetchDownloaderStatus();
        downloaderAvailable = Boolean(status.available);

        if (!downloaderAvailable) {
            alertBox.style.display = 'flex';
            if (status.message) {
                alertMsg.innerHTML = `${escapeHtml(status.message)}`;
            }
            if (submitBtn) {
                submitBtn.title = 'Ferramentas de download não instaladas no servidor.';
            }
        } else {
            alertBox.style.display = 'none';
            if (submitBtn) {
                submitBtn.title = 'Baixar vídeo';
            }
        }
    } catch (err) {
        console.warn('Erro ao verificar status das ferramentas:', err);
    }
}

function setupUrlDownloader() {
    const form = document.getElementById('url-download-form');
    const input = document.getElementById('url-download-input');
    const submitBtn = document.getElementById('btn-submit-download');
    const btnIcon = document.getElementById('btn-download-icon');
    const btnLabel = document.getElementById('btn-download-label');
    const statusText = document.getElementById('download-status-text');
    const progressBox = document.getElementById('download-progress-box');
    const progressStatus = document.getElementById('download-progress-status');
    const progressPercent = document.getElementById('download-progress-percent');
    const progressFill = document.getElementById('download-progress-fill');

    if (!form) return;

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const url = input.value.trim();
        if (!url) return;

        if (!downloaderAvailable) {
            alert('Atenção: As ferramentas necessárias para download de vídeos não estão instaladas no servidor.\nExecute "sudo ./install-video-tools.sh" no servidor para habilitar esta funcionalidade.');
            return;
        }

        // Estado de carregamento com barra de progresso
        input.disabled = true;
        submitBtn.disabled = true;
        btnIcon.textContent = '⏳';
        btnLabel.textContent = 'Baixando...';
        if (statusText) {
            statusText.style.display = 'none';
            statusText.textContent = '';
        }
        if (progressBox) {
            progressBox.style.display = 'block';
            progressStatus.textContent = 'Iniciando download...';
            progressPercent.textContent = '0%';
            progressFill.style.width = '0%';
            progressFill.classList.remove('processing');
        }

        try {
            const res = await downloadVideoWithProgress(url, (evt) => {
                if (evt.status === 'downloading') {
                    const pct = Math.round(evt.percent || 0);
                    if (progressFill) {
                        progressFill.classList.remove('processing');
                        progressFill.style.width = `${pct}%`;
                    }
                    if (progressPercent) progressPercent.textContent = `${pct}%`;
                    if (progressStatus) progressStatus.textContent = `Baixando vídeo... ${pct}%`;
                } else if (evt.status === 'processing') {
                    if (progressFill) {
                        progressFill.classList.add('processing');
                        progressFill.style.width = '100%';
                    }
                    if (progressPercent) progressPercent.textContent = '';
                    if (progressStatus) progressStatus.textContent = 'Processando...';
                }
            });

            if (progressFill) progressFill.style.width = '100%';
            if (progressPercent) progressPercent.textContent = '100%';
            if (progressStatus) progressStatus.textContent = 'Download concluído!';

            if (statusText) {
                statusText.style.display = 'block';
                statusText.textContent = `✓ Vídeo "${res.name}" salvo com sucesso!`;
            }
            showToast(`Vídeo "${res.name}" pronto para exibição!`);
            input.value = '';

            await loadMediaData();
            if (res.name) {
                selectItem({ name: res.name, type: 'VIDEO' });
            }

            setTimeout(() => {
                if (progressBox) progressBox.style.display = 'none';
            }, 3000);
        } catch (err) {
            alert(`Erro no download: ${err.message}`);
            if (progressBox) progressBox.style.display = 'none';
            if (statusText) {
                statusText.style.display = 'block';
                statusText.textContent = `Erro: ${err.message}`;
            }
        } finally {
            input.disabled = false;
            submitBtn.disabled = false;
            btnIcon.textContent = '📥';
            btnLabel.textContent = 'Baixar Vídeo';
        }
    });
}

function setupNavLinks() {
    document.querySelectorAll('.header-nav a.nav-link').forEach(link => {
        const href = link.getAttribute('href');
        if (href && href.startsWith('/') && !href.startsWith('/presenter')) {
            link.href = withBasePath(href);
        }
    });
}

document.addEventListener('DOMContentLoaded', () => {
    setupNavLinks();
    setupUploadAndDropZone();
    setupUrlDownloader();
    setupControlButtons();
    setupTabsAndSearch();
    loadMediaData();
    loadCurrentContent();
    checkDownloaderToolsStatus();

    // Sincroniza status do telão periodicamente
    setInterval(loadCurrentContent, 2000);
});
