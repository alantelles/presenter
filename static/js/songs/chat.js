const CHAT_PROTOCOL = 'presenter-chat';
const MAX_RENDERED_MESSAGES = 100;
export const CHAT_NAME_STORAGE_KEY = 'presenter.chat.username';
const MAX_CHAT_NAME_LENGTH = 40;
let activeSocket;

export function sendSongLoadCommand(songName) {
    if (!activeSocket || activeSocket.readyState !== WebSocket.OPEN) {
        return false;
    }
    activeSocket.send(JSON.stringify({ type: 'command', action: 'loadSongByName', songName }));
    return true;
}

export function initializeChat() {
    const dialog = document.getElementById('chat-dialog');
    const nameDialog = document.getElementById('chat-name-dialog');
    const nameInput = document.getElementById('chat-name-input');
    const nameForm = document.getElementById('chat-name-form');
    const nameCancel = document.getElementById('chat-name-cancel');
    const triggers = document.querySelectorAll('.chat-trigger');
    const closeButton = document.getElementById('chat-close');
    const status = document.getElementById('chat-status');
    const messages = document.getElementById('chat-messages');
    const form = document.getElementById('chat-form');
    const input = document.getElementById('chat-input');
    const unreadBadges = document.querySelectorAll('.chat-unread');
    const authToken = document.getElementById('basic-auth-token').value;
    let unreadCount = 0;
    let reconnectDelay = 1000;
    let socket;
    let username = '';

    function isValidUsername(value) {
        return value.length > 0 && [...value].length <= MAX_CHAT_NAME_LENGTH && !/[\r\n]/.test(value);
    }

    function showNameDialog() {
        if (!nameDialog.open) {
            nameDialog.showModal();
            nameInput.focus();
        }
    }

    function setStatus(message, state) {
        status.textContent = message;
        status.dataset.state = state;
    }

    function updateUnread() {
        unreadBadges.forEach(badge => {
            badge.textContent = unreadCount > 99 ? '99+' : String(unreadCount);
            badge.hidden = unreadCount === 0;
        });
    }

    function scrollToLatestMessage() {
        window.requestAnimationFrame(() => {
            messages.scrollTop = messages.scrollHeight;
        });
    }

    function focusMessageInputOnDesktop() {
        if (!window.matchMedia('(max-width: 1000px)').matches) {
            input.focus();
        }
    }

    function renderMessage(message) {
        const item = document.createElement('li');
        item.className = 'chat-message';

        const metadata = document.createElement('div');
        metadata.className = 'chat-message-meta';
        const username = document.createElement('strong');
        username.textContent = message.username;
        const time = document.createElement('time');
        const sentAt = new Date(message.sentAt);
        time.dateTime = sentAt.toISOString();
        time.textContent = sentAt.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        metadata.append(username, time);

        const content = document.createElement('p');
        content.className = 'chat-message-content';
        content.textContent = message.content;
        item.append(metadata, content);
        messages.append(item);

        while (messages.children.length > MAX_RENDERED_MESSAGES) {
            messages.firstElementChild.remove();
        }
        messages.scrollTop = messages.scrollHeight;
    }

    function connect() {
        const isPrefixed = window.location.pathname.startsWith('/presenter');
        const endpoint = new URL(`${isPrefixed ? '/presenter' : ''}/api/chat`, window.location.href);
        endpoint.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const encodedCredential = authToken.replace(/=/g, '').replace(/\+/g, '-').replace(/\//g, '_');
        socket = new WebSocket(endpoint, [CHAT_PROTOCOL, `auth.${encodedCredential}`]);
        activeSocket = socket;

        socket.addEventListener('open', () => {
            reconnectDelay = 1000;
            socket.send(JSON.stringify({ type: 'identify', username }));
            setStatus(`Conectado como ${username}`, 'connected');
        });
        socket.addEventListener('message', event => {
            try {
                const message = JSON.parse(event.data);
                renderMessage(message);
                if (message.type === 'system' && message.action === 'refreshSongs' && !message.history) {
                    window.dispatchEvent(new CustomEvent('presenter:refresh-songs', { detail: message }));
                }
                if (message.type === 'system' && message.action === 'loadSongByName' && !message.history && validSongName(message.songName)) {
                    window.dispatchEvent(new CustomEvent('presenter:load-song-by-name', { detail: message }));
                }
                if (!message.history && !dialog.open) {
                    unreadCount += 1;
                    updateUnread();
                    window.dispatchEvent(new CustomEvent('presenter:new-chat-message', { detail: message }));
                }
            } catch (error) {
                console.error('Não foi possível ler uma mensagem do chat:', error);
            }
        });
        socket.addEventListener('close', () => {
            setStatus('Conexão perdida. Reconectando...', 'disconnected');
            window.setTimeout(connect, reconnectDelay);
            reconnectDelay = Math.min(reconnectDelay * 2, 30000);
        });
        socket.addEventListener('error', () => socket.close());
    }

    function validSongName(songName) {
        return typeof songName === 'string' && songName.length > 0 && !/[\\/]/.test(songName) && songName.endsWith('.txt');
    }

    triggers.forEach(trigger => {
        trigger.addEventListener('click', () => {
            if (!username) {
                showNameDialog();
                return;
            }
            if (!dialog.open) {
                dialog.showModal();
                scrollToLatestMessage();
                unreadCount = 0;
                updateUnread();
                focusMessageInputOnDesktop();
            }
        });
    });
    closeButton.addEventListener('click', () => dialog.close());
    nameCancel.addEventListener('click', () => nameDialog.close());
    nameForm.addEventListener('submit', event => {
        event.preventDefault();
        const enteredName = nameInput.value.trim();
        if (!isValidUsername(enteredName)) {
            nameInput.setCustomValidity('Informe um nome com até 40 caracteres.');
            nameInput.reportValidity();
            return;
        }
        nameInput.setCustomValidity('');
        username = enteredName;
        localStorage.setItem(CHAT_NAME_STORAGE_KEY, username);
        nameDialog.close();
        connect();
    });
    form.addEventListener('submit', event => {
        event.preventDefault();
        const content = input.value.trim();
        if (!content || !socket || socket.readyState !== WebSocket.OPEN) {
            return;
        }
        socket.send(JSON.stringify({ type: 'message', content }));
        input.value = '';
        focusMessageInputOnDesktop();
    });

    const savedName = localStorage.getItem(CHAT_NAME_STORAGE_KEY)?.trim() || '';
    if (isValidUsername(savedName)) {
        username = savedName;
        connect();
    } else {
        localStorage.removeItem(CHAT_NAME_STORAGE_KEY);
        showNameDialog();
    }
}