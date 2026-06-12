const palette = [
  'oklch(0.55 0.14 192)', // Teal
  'oklch(0.52 0.16 280)', // Violet
  'oklch(0.60 0.15 75)',  // Amber
  'oklch(0.55 0.17 15)',  // Rose
  'oklch(0.57 0.14 155)', // Emerald
  'oklch(0.50 0.16 255)', // Indigo
  'oklch(0.58 0.16 30)',  // Coral
  'oklch(0.48 0.05 230)', // Slate
]

export function useAvatarColor(userId: number): string {
  return palette[userId % palette.length]
}
