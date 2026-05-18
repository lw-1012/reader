import { api } from './api.js';

let current = null;

export async function playTTS(text, voice) {
  if (!text || !text.trim()) return;
  try {
    if (current) { current.pause(); current = null; }
    const audio = new Audio(api.ttsUrl(text.trim(), voice));
    audio.preload = 'auto';
    current = audio;
    await audio.play();
  } catch (e) {
    console.error('tts play', e);
  }
}

export function stopTTS() {
  if (current) { try { current.pause(); } catch {} current = null; }
}
