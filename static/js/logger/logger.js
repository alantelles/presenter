import {
    saveMedia,
    moveMedia,
    fetchSongContent,
} from './client.js';

document.addEventListener('DOMContentLoaded', () => {
  const buttons = document.querySelectorAll('.logger-momento');

  buttons.forEach(button => {
    button.addEventListener('click', () => {
      const momento = button.value;
      const timestamp = new Date().toISOString();
      saveLog(momento, timestamp);
      // Aqui você pode adicionar a lógica para enviar os dados para o servidor, se necessário
    });
  });
  async function saveLog(momento, timestamp) {
    const dataYYYYMMDD = timestamp.split('T')[0].replace(/-/g, '');
    const title = `log_${dataYYYYMMDD}`;
    const author = 'logger';
    const mediaId = `${title} - ${author}.txt`;
    const atual = await fetchSongContent('logsCulto/' + mediaId);
    const category = 'songs';
    const prefix = atual ? atual + '\n' : 'timestamp|momento\n';
    const entry = {
      title,
      author,
      category,
      content: `${prefix}${timestamp}|${momento}`,
    }
    await saveMedia(entry);
    try {
      const body = {
        mediaId,
        category,
        destination: 'logsCulto'
    };
      await moveMedia(body);
    } catch (error) {
      console.error('Error moving log:', error);
    }
    console.log('Log saved and moved successfully.');    
  }
});