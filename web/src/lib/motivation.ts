/*
 * Presentation for the four motivation bands. The copy itself (label, title,
 * message) is NOT here: the API computes it from live progress and returns it on
 * every topic as `topic.motivation` (source of truth: internal/domain/motivation.go).
 * The client only maps each band to its tone — the colour of its state label.
 */
import type { components } from '../api/schema'

export type Motivation = components['schemas']['Motivation']
export type MotivationBand = Motivation['band']

const TONE_CLASS: Record<MotivationBand, string> = {
  'not-started': 'text-text-muted',
  'in-progress': 'text-blue-ink',
  'almost-there': 'text-teal-ink',
  achieved: 'text-gold-ink',
}

/** Tailwind text-colour utility for a band's tone (muted · blue · teal · gold). */
export function bandToneClass(band: MotivationBand): string {
  return TONE_CLASS[band]
}
