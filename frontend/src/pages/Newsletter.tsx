import { useEffect, useRef, useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { api } from '@/api';
import { Seo } from '@/components/ui/Seo';

type State = 'working' | 'done' | 'failed';

const COPY = {
  confirm: {
    title: 'Confirm your subscription',
    working: 'Confirming your subscription…',
    done: 'You are subscribed. Your welcome code is on its way to your inbox.',
    failed: 'This confirmation link is invalid or has already been used.',
  },
  unsubscribe: {
    title: 'Unsubscribe',
    working: 'Unsubscribing you…',
    done: 'You have been unsubscribed and will not receive any more newsletters from us.',
    failed: 'This unsubscribe link is invalid. Reply to any newsletter and we will remove you by hand.',
  },
} as const;

/** Landing page for the links in newsletter emails (double opt-in and unsubscribe). */
export default function Newsletter() {
  const { action } = useParams();
  const [params] = useSearchParams();
  const kind = action === 'unsubscribe' ? 'unsubscribe' : 'confirm';
  const copy = COPY[kind];
  const [state, setState] = useState<State>('working');
  const sent = useRef(false);

  useEffect(() => {
    if (sent.current) return; // StrictMode runs effects twice in development
    sent.current = true;
    const token = params.get('token') ?? '';
    const call = kind === 'confirm' ? api.confirmNewsletter : api.unsubscribeNewsletter;
    if (!token) {
      setState('failed');
      return;
    }
    call(token).then(
      () => setState('done'),
      () => setState('failed'),
    );
  }, [kind, params]);

  return (
    <main className="container page" style={{ textAlign: 'center', paddingTop: 80, paddingBottom: 96 }}>
      <Seo title={copy.title} noindex />
      <h1 className="page-title">{copy.title}</h1>
      <p className="page-intro" role="status" aria-live="polite">
        {copy[state]}
      </p>
      <div style={{ marginTop: 28 }}>
        <Link to="/" className="btn btn--primary">
          Back to home
        </Link>
      </div>
    </main>
  );
}
