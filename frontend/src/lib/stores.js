import { writable } from 'svelte/store';

export const route = writable(parseHash());
export const toast = writable(null);

function parseHash() {
  const h = location.hash.replace(/^#/, '') || '/';
  const [path, query] = h.split('?');
  const parts = path.split('/').filter(Boolean);
  return { path, parts, query: new URLSearchParams(query || '') };
}

window.addEventListener('hashchange', () => route.set(parseHash()));

export function go(path) {
  if (location.hash !== '#' + path) location.hash = path;
}

let toastTimer;
export function showToast(message, ms = 2200) {
  toast.set(message);
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toast.set(null), ms);
}
