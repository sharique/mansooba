// A drag that ends over a card can be followed by a click on it. The column marks the
// end of every drag and the card ignores a click that comes straight after, so dragging
// a card never also opens the task. A browser drag does not click, but the touch
// fallback of the drag library may.
let lastDragEnd = 0

export function markDragEnd(now: number = Date.now()): void {
  lastDragEnd = now
}

export function wasJustDragged(now: number = Date.now(), windowMs = 100): boolean {
  return now - lastDragEnd < windowMs
}
