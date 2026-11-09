const CHAT_PROTOCOL = 'presenter-chat';
const MAX_RENDERED_MESSAGES = 100;

export function initializeChat() {
    const dialog = document.getElementById('chat-dialog');
    const trigger = document.getElementById('chat-trigger');
    const closeButton = document.getElementById('chat-close');
    const status = document.getElementById('chat-status');
    const messages = document.getElementById('chat-messages');
    const form = document.getElementById('chat-form');
    const input = document.getElementById('chat-input');
    const unreadBadge = document.getElementById('chat-unread');
    const authToken = document.getElementById('basic-auth-token').value;
    let unreadCount = 0;
    let reconnectDelay = 1000;
    let socket;

    function setStatus(message, state) {
        status.textContent = message;
        status.dataset.state = state;
    }

    function updateUnread() {
        unreadBadge.textContent = unreadCount > 99 ? '99+' : String(unreadCount);
        unreadBadge.hidden = unreadCount === 0;
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

        socket.addEventListener('open', () => {
            reconnectDelay = 1000;
            setStatus('Conectado', 'connected');
        });
        socket.addEventListener('message', event => {
            try {
                const message = JSON.parse(event.data);
                renderMessage(message);
                if (!message.history && !dialog.open) {
                    unreadCount += 1;
                    updateUnread();
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

    trigger.addEventListener('click', () => {
        if (!dialog.open) {
            dialog.showModal();
            unreadCount = 0;
            updateUnread();
            input.focus();
        }
    });
    closeButton.addEventListener('click', () => dialog.close());
    form.addEventListener('submit', event => {
        event.preventDefault();
        const content = input.value.trim();
        if (!content || socket.readyState !== WebSocket.OPEN) {
            return;
        }
        socket.send(JSON.stringify({ content }));
        input.value = '';
        input.focus();
    });

    connect();
}