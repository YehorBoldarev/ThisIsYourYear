import { useState } from 'react'
import type { ReactNode } from 'react'
import {
  ACCENTS,
  Avatar,
  Button,
  Chip,
  Confetti,
  Icon,
  type IconName,
  Input,
  Label,
  Logo,
  Modal,
  ProgressBar,
  Ring,
  SwatchGroup,
  Textarea,
  Toggle,
  type AccentColor,
} from '../components'

const ICONS: IconName[] = [
  'check',
  'star',
  'x',
  'plus',
  'grid',
  'pencil',
  'sun',
  'moon',
  'dot',
  'arrow-right',
]

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded-card border border-border bg-elevated p-7 shadow-card">
      <h2 className="label-mono mb-5 text-[11px] text-text-muted">{title}</h2>
      <div className="flex flex-wrap items-center gap-4">{children}</div>
    </section>
  )
}

/** Dev-only gallery: every primitive in every state, for cross-theme QA. */
export function Gallery() {
  const [accent, setAccent] = useState<AccentColor>('teal')
  const [celebrate, setCelebrate] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)

  return (
    <div className="mx-auto w-full max-w-[960px] space-y-6 px-6 py-10">
      <header>
        <div className="label-mono text-[11px] text-text-muted">Dev · Component gallery</div>
        <h1 className="mt-2 font-serif text-3xl font-medium text-text">Design system</h1>
        <p className="mt-2 text-sm text-text-2">
          Flip the theme in the header and resize to 768 / 390 — everything should stay on-brand.
        </p>
      </header>

      <Section title="Logo (year progress)">
        <Logo progress={0.15} size={44} />
        <Logo progress={0.48} size={44} />
        <Logo progress={0.85} size={44} />
        <Logo progress={0.48} withWordmark />
      </Section>

      <Section title="Buttons">
        <Button variant="teal">Create goal</Button>
        <Button variant="gold">
          <Icon name="star" size={16} /> Mark achieved
        </Button>
        <Button variant="outline">Edit goal</Button>
        <Button variant="teal" disabled>
          Disabled
        </Button>
        <Button variant="teal" size="sm">
          Small
        </Button>
      </Section>

      <Section title="Fields">
        <div className="w-full max-w-xs space-y-2">
          <Label htmlFor="g-title">What is the goal?</Label>
          <Input id="g-title" placeholder="Run a half-marathon" />
        </div>
        <div className="w-full max-w-xs space-y-2">
          <Label htmlFor="g-title-err">With error</Label>
          <Input id="g-title-err" invalid defaultValue="Too short" />
        </div>
        <div className="w-full max-w-sm space-y-2">
          <Label htmlFor="g-why">Why does it matter?</Label>
          <Textarea id="g-why" placeholder="Because…" />
        </div>
      </Section>

      <Section title="Progress bars">
        <div className="w-full max-w-md space-y-4">
          <ProgressBar value={0.48} height={8} gradient label="Year" />
          <ProgressBar value={0.75} height={6} accent="teal" label="Goal" />
          <ProgressBar value={0.33} height={6} accent="blue" label="Goal" />
          <ProgressBar value={1} height={6} accent="gold" label="Goal" />
        </div>
      </Section>

      <Section title="Rings">
        <Ring value={0.25} />
        <Ring value={0.75} accent="blue" />
        <Ring value={1} accent="gold" />
      </Section>

      <Section title="Confetti (celebration; respects reduced-motion)">
        <div className="relative h-28 w-56 overflow-hidden rounded-field border border-border bg-surface">
          <Confetti />
        </div>
      </Section>

      <Section title="Icons">
        {ICONS.map((name) => (
          <span key={name} className="flex flex-col items-center gap-1 text-text-2">
            <Icon name={name} title={name} />
            <span className="label-mono text-[9px] text-text-muted">{name}</span>
          </span>
        ))}
      </Section>

      <Section title="Chips">
        <Chip>Run a marathon</Chip>
        <Chip>Read 24 books</Chip>
        <Chip dot={accent}>Selected accent</Chip>
      </Section>

      <Section title="Toggle & swatches">
        <div className="flex items-center gap-3">
          <Toggle checked={celebrate} onChange={setCelebrate} label="Celebrate as an achievement" />
          <span className="text-sm text-text-2">Celebrate as an achievement ★</span>
        </div>
        <SwatchGroup value={accent} onChange={setAccent} options={ACCENTS} />
      </Section>

      <Section title="Avatar">
        <Avatar initials="NA" />
        <Avatar initials="YE" size={48} />
      </Section>

      <Section title="Modal">
        <Button onClick={() => setModalOpen(true)}>Open modal</Button>
        <Modal
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          eyebrow="New for 2026"
          title="Create a big goal"
          footer={
            <>
              <Button variant="outline" onClick={() => setModalOpen(false)}>
                Cancel
              </Button>
              <Button onClick={() => setModalOpen(false)}>Create goal</Button>
            </>
          }
        >
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="m-title">What is the goal?</Label>
              <Input id="m-title" placeholder="Run a half-marathon" />
            </div>
            <div className="space-y-2">
              <Label>Accent colour</Label>
              <SwatchGroup value={accent} onChange={setAccent} />
            </div>
          </div>
        </Modal>
      </Section>
    </div>
  )
}
