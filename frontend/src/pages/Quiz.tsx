import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Seo } from '@/components/ui/Seo';
import type { ScentFamily, Wear } from '@/types';

/**
 * Five-question scent finder. Scoring is client-side and deliberately simple:
 * each answer adds weight to scent families; the result links to a filtered
 * collection so the real catalogue (and the Go API) does the matching.
 */
interface Q {
  q: string;
  options: Array<{ label: string; families: ScentFamily[]; wear?: Wear[] }>;
}

const QUESTIONS: Q[] = [
  {
    q: 'When will you wear it most?',
    options: [
      { label: 'Daytime and work', families: ['fresh-musk', 'floral'], wear: ['day'] },
      { label: 'Evenings and nights out', families: ['woody-oud', 'amber-oriental'], wear: ['evening'] },
      { label: 'Special occasions', families: ['woody-oud', 'spicy'], wear: ['evening'] },
    ],
  },
  {
    q: 'Which season suits you?',
    options: [
      { label: 'Spring / summer', families: ['fresh-musk', 'floral'], wear: ['summer'] },
      { label: 'Autumn / winter', families: ['amber-oriental', 'woody-oud', 'gourmand'], wear: ['winter'] },
      { label: 'All year', families: ['woody-oud', 'floral'] },
    ],
  },
  {
    q: 'Pick a mood',
    options: [
      { label: 'Warm and sweet', families: ['gourmand', 'amber-oriental'] },
      { label: 'Dark and smoky', families: ['woody-oud', 'spicy'] },
      { label: 'Clean and soft', families: ['fresh-musk', 'floral'] },
    ],
  },
  {
    q: 'How strong should it be?',
    options: [
      { label: 'Noticeable but close to the skin', families: ['fresh-musk', 'floral'] },
      { label: 'Fills the room', families: ['woody-oud', 'spicy', 'amber-oriental'] },
    ],
  },
  {
    q: 'Which note draws you in?',
    options: [
      { label: 'Rose', families: ['floral'] },
      { label: 'Oud', families: ['woody-oud'] },
      { label: 'Vanilla', families: ['gourmand', 'amber-oriental'] },
      { label: 'Saffron', families: ['spicy'] },
    ],
  },
];

const FAMILY_NAME: Record<ScentFamily, string> = {
  'woody-oud': 'Woody & Oud',
  'amber-oriental': 'Amber & Oriental',
  floral: 'Floral',
  'fresh-musk': 'Fresh & Musk',
  gourmand: 'Gourmand',
  spicy: 'Spicy',
};

export default function Quiz() {
  const [step, setStep] = useState(0);
  const [answers, setAnswers] = useState<number[]>([]);
  const done = step >= QUESTIONS.length;

  const result = () => {
    const score: Partial<Record<ScentFamily, number>> = {};
    const wear = new Set<Wear>();
    answers.forEach((a, i) => {
      const opt = QUESTIONS[i].options[a];
      opt.families.forEach((f) => (score[f] = (score[f] ?? 0) + 1));
      opt.wear?.forEach((w) => wear.add(w));
    });
    const top = (Object.entries(score) as Array<[ScentFamily, number]>).sort((a, b) => b[1] - a[1]).slice(0, 2);
    const params = new URLSearchParams();
    params.set('family', top.map(([f]) => f).join(','));
    if (wear.size) params.set('wear', [...wear].join(','));
    return { top, href: `/collections/all?${params.toString()}` };
  };

  return (
    <main className="container container--narrow page" style={{ maxWidth: 760 }}>
      <Seo title="Find your scent" />
      <nav aria-label="Breadcrumb" className="breadcrumb">
        <Link to="/">Home</Link> › <span>Scent quiz</span>
      </nav>
      <h1 className="page-title">Find your scent in 60 seconds</h1>
      <p className="page-intro">Answer {QUESTIONS.length} quick questions and we'll point you to your best matches.</p>

      {!done ? (
        <section className="card" style={{ marginTop: 24, padding: 32, display: 'flex', flexDirection: 'column', gap: 18 }} aria-live="polite">
          <span className="eyebrow">
            Question {step + 1} of {QUESTIONS.length}
          </span>
          <h2 style={{ fontSize: 30 }}>{QUESTIONS[step].q}</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {QUESTIONS[step].options.map((o, i) => (
              <button
                key={o.label}
                type="button"
                className="method"
                onClick={() => {
                  setAnswers((a) => [...a.slice(0, step), i]);
                  setStep(step + 1);
                }}
              >
                <b>{o.label}</b>
              </button>
            ))}
          </div>
          {step > 0 && (
            <button type="button" className="btn btn--ghost" style={{ alignSelf: 'flex-start' }} onClick={() => setStep(step - 1)}>
              ← Back
            </button>
          )}
        </section>
      ) : (
        (() => {
          const r = result();
          return (
            <section className="panel" style={{ marginTop: 24, padding: 32, display: 'flex', flexDirection: 'column', gap: 16 }}>
              <span className="eyebrow">Your match</span>
              <h2 style={{ fontSize: 32 }}>{r.top.map(([f]) => FAMILY_NAME[f]).join(' & ')}</h2>
              <p className="muted">We've filtered the collection to the fragrances that fit your answers. Try a 2ml sample of each before you commit.</p>
              <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
                <Link to={r.href} className="btn btn--primary btn--lg">
                  See my matches
                </Link>
                <Link to="/products/discovery-set" className="btn btn--secondary btn--lg">
                  Try the Discovery Set
                </Link>
              </div>
              <button
                type="button"
                className="btn btn--ghost"
                style={{ alignSelf: 'flex-start' }}
                onClick={() => {
                  setStep(0);
                  setAnswers([]);
                }}
              >
                Start again
              </button>
            </section>
          );
        })()
      )}
    </main>
  );
}
