import { uploadImage, fetchImageList, emitImageContent, clearMainContent, thumbUrl } from './client.js';

async function renderGallery() {
    const gallery = document.getElementById('gallery');
    gallery.innerHTML = '';
    let names;
    try {
        names = await fetchImageList();
    } catch (err) {
        gallery.textContent = `Erro ao carregar imagens: ${err.message}`;
        return;
    }
    for (const name of names) {
        const item = document.createElement('button');
        item.className = 'gallery-item';
        item.type = 'button';
        item.title = name;

        const img = document.createElement('img');
        img.src = thumbUrl(name);
        img.alt = name;
        img.onerror = () => { img.replaceWith(document.createTextNode(name)); };

        item.appendChild(img);
        item.addEventListener('click', () => emitImageContent(name));
        gallery.appendChild(item);
    }
}

function setupUploadForm() {
    const form = document.getElementById('upload-form');
    const input = document.getElementById('file-input');
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        if (!input.files.length) {
            return;
        }
        try {
            await uploadImage(input.files[0]);
            input.value = '';
            await renderGallery();
        } catch (err) {
            alert(err.message);
        }
    });
}

function setupClearButton() {
    document.getElementById('clear-main').addEventListener('click', () => {
        clearMainContent();
    });
}

document.addEventListener('DOMContentLoaded', () => {
    setupUploadForm();
    setupClearButton();
    renderGallery();
});
