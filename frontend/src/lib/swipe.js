export function swipe(node, opts = {}) {
  let threshold = opts.threshold ?? 60;
  let startX = 0, startY = 0, t0 = 0, active = false, locked = false;

  function start(e) {
    const t = e.touches ? e.touches[0] : e;
    startX = t.clientX; startY = t.clientY; t0 = Date.now();
    active = true; locked = false;
  }
  function move(e) {
    if (!active) return;
    const t = e.touches ? e.touches[0] : e;
    const dx = t.clientX - startX;
    const dy = t.clientY - startY;
    if (!locked && (Math.abs(dx) > 8 || Math.abs(dy) > 8)) {
      locked = Math.abs(dx) > Math.abs(dy) ? 'x' : 'y';
    }
  }
  function end(e) {
    if (!active) return;
    active = false;
    const t = e.changedTouches ? e.changedTouches[0] : e;
    const dx = t.clientX - startX;
    const dy = t.clientY - startY;
    const dt = Date.now() - t0;
    if (locked === 'x' && Math.abs(dx) > threshold && dt < 700) {
      node.dispatchEvent(new CustomEvent(dx > 0 ? 'swiperight' : 'swipeleft'));
    }
  }
  node.addEventListener('touchstart', start, { passive: true });
  node.addEventListener('touchmove',  move,  { passive: true });
  node.addEventListener('touchend',   end);
  node.addEventListener('touchcancel', end);
  return {
    destroy() {
      node.removeEventListener('touchstart', start);
      node.removeEventListener('touchmove',  move);
      node.removeEventListener('touchend',   end);
      node.removeEventListener('touchcancel', end);
    }
  };
}
