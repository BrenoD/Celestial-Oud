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
          <div className="accordion__item" key={it.id}>
            <button
              type="button"
              className="accordion__trigger"
              aria-expanded={isOpen}
              aria-controls={panelId}
              onClick={() => setOpen(isOpen ? null : it.id)}
            >
              {it.title}
              <span aria-hidden="true" className="accordion__sign">
                {isOpen ? '−' : '+'}
              </span>
            </button>
            {isOpen && (
              <div id={panelId} className="accordion__panel">
                {it.body}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
