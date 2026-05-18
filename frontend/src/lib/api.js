async function request(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' }, credentials: 'include' };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const r = await fetch(path, opts);
  if (r.status === 401) {
    if (location.hash !== '#/login') location.hash = '#/login';
    throw new Error('unauthorized');
  }
  const ct = r.headers.get('content-type') || '';
  const data = ct.includes('application/json') ? await r.json().catch(() => null) : await r.text();
  if (!r.ok) throw new Error(typeof data === 'string' ? data : (data && data.error) || 'request failed');
  return data;
}

export const api = {
  // auth
  login: (password) => request('POST', '/api/auth/login', { password }),
  logout: () => request('POST', '/api/auth/logout'),
  check: () => request('GET', '/api/auth/check'),

  // settings
  getSettings: () => request('GET', '/api/settings'),
  saveSettings: (patch) => request('PUT', '/api/settings', patch),

  // books
  listBooks: () => request('GET', '/api/books'),
  getBook: (id) => request('GET', `/api/books/${id}`),
  deleteBook: (id) => request('DELETE', `/api/books/${id}`),
  importBook: async (file) => {
    const text = typeof file === 'string' ? file : await file.text();
    const obj = JSON.parse(text);
    return request('POST', '/api/books/import', obj);
  },

  // reading
  getParagraphs: (bookId, from, to) =>
    request('GET', `/api/books/${bookId}/paragraphs?from=${from}&to=${to}`),
  simplify: (paragraphId, level, force = false) =>
    request('POST', `/api/paragraphs/${paragraphId}/simplify?level=${encodeURIComponent(level)}${force ? '&force=1' : ''}`),
  analyze: (text, level) => request('POST', '/api/analyze', { text, level }),

  // progress
  getProgress: (bookId) => request('GET', `/api/books/${bookId}/progress`),
  putProgress: (bookId, paragraphId, level) =>
    request('PUT', `/api/books/${bookId}/progress`, { paragraph_id: paragraphId, level }),

  // tts
  ttsUrl: (text, voice) => `/api/tts?text=${encodeURIComponent(text)}${voice ? `&voice=${encodeURIComponent(voice)}` : ''}`
};
