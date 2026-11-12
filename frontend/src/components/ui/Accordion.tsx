import { useState, type ReactNode } from 'react';
import './Accordion.css';

export interface AccordionItem {
  id: string;
  title: string;
  body: ReactNode;
}

interface Props {
  items: AccordionItem[];
  defaultOpen?: string | null;
  size?: 'md' | 'lg';
}

export function Accordion({ items, defaultOpen = null, size = 'md' }: Props) {
  const [open, setOpen] = useState<string | null>(defaultOpen);
  return (
    <div className={`accordion accordion--${size}`}>
      {items.map((it) => {
        const isOpen = open === it.id;
        const panelId = `acc-${it.id}`;
        return (
          <div className={`accordion__item${isOpen ? ' is-open' : ''}`} key={it.id}>
            <button
              type="button"
              className="accordion__trigger"
              aria-expanded={isOpen}
              aria-controls={panelId}
              onClick={() => setOpen(isOpen ? null : it.id)}
            >
              {it.title}
              <span aria-hidden="true" className="accordion__sign" />
            </button>
            {/* Always rendered so it can animate open; inert while closed (not focusable or read out). */}
            <div id={panelId} className="accordion__panel" role="region" aria-label={it.title} {...(isOpen ? {} : { inert: '' })}>
              <div className="accordion__inner">{it.body}</div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
