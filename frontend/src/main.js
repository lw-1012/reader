import './app.css';
import App from './App.svelte';

const app = new App({ target: document.getElementById('app') });
export default app;

// register a tiny service worker for offline shell (only over https)
if ('serviceWorker' in navigator && location.protocol === 'https:') {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {});
  });
}
