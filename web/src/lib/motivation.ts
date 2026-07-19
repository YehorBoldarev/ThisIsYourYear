/*
 * The single source of truth for a goal's motivation band. Maps a completion
 * ratio to one of four bands carrying the design's canonical copy (screens 3 & 7):
 * a mono short label, a tone (colour class), a Newsreader title, and an italic
 * message. Used by the dashboard chapter rows, the topic-detail callout, and the
 * motivation reference — so every "how is this going" reads the same everywhere.
 */
export type MotivationBand = 'not-started' | 'in-progress' | 'almost-there' | 'achieved'
export type BandTone = 'muted' | 'blue' | 'teal' | 'gold'

export interface BandInfo {
  band: MotivationBand
  tone: BandTone
  /** Tailwind text-colour utility for the tone. */
  toneClass: string
  /** Mono state label, e.g. "Almost there". */
  shortLabel: string
  /** Newsreader title, e.g. "In sight". */
  title: string
  /** The italic encouragement shown in callouts. */
  message: string
}

const BANDS: Record<MotivationBand, Omit<BandInfo, 'band'>> = {
  'not-started': {
    tone: 'muted',
    toneClass: 'text-text-muted',
    shortLabel: 'Just started',
    title: 'The blank page',
    message:
      'A blank page isn’t empty — it’s full of everything that could be. Make the first mark today.',
  },
  'in-progress': {
    tone: 'blue',
    toneClass: 'text-blue',
    shortLabel: 'In progress',
    title: 'Building momentum',
    message:
      'Momentum is quiet. Keep showing up, and the small days compound into a remarkable year.',
  },
  'almost-there': {
    tone: 'teal',
    toneClass: 'text-teal',
    shortLabel: 'Almost there',
    title: 'In sight',
    message:
      'You can see it from here. Don’t trade the finish line for comfort — finish what you started.',
  },
  achieved: {
    tone: 'gold',
    toneClass: 'text-gold',
    shortLabel: 'Achieved',
    title: 'A promise kept',
    message:
      'You kept a promise to yourself. This is exactly what that feels like. Take a moment to savour it.',
  },
}

/** Ratio (0–1) + goal count → the motivation band. `total===0` is "not started". */
export function motivationBand(ratio: number, total: number): BandInfo {
  const band: MotivationBand =
    total === 0 || ratio <= 0
      ? 'not-started'
      : ratio >= 1
        ? 'achieved'
        : ratio >= 0.8
          ? 'almost-there'
          : 'in-progress'
  return { band, ...BANDS[band] }
}
