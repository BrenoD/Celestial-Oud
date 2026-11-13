import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import { api } from '@/api';
import { ApiError } from '@/api/client';
import { useAuthStore } from '@/store/auth';
import { useOverlay } from '@/lib/useOverlay';
import { usePolling } from '@/lib/usePolling';
import type { ChatMessage } from '@/types';
import { loadChatRef, saveChatRef, type ChatRef } from './chatStore';

const POLL_MS = 4_000;

const time = (iso: string) => new Date(iso).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });

function Bubble({ m }: { m: ChatMessage }) {
  return (
    <li className={`chat-msg chat-msg--${m.sender}`}>
      {m.sender === 'agent' && <span className="chat-msg__who">Celestial Oud</span>}
      {/* Rendered as text: React escapes it, so messages can never inject markup. */}
      <p className="chat-msg__body">{m.body}</p>
      <time className="chat-msg__time" dateTime={m.createdAt}>
        {time(m.createdAt)}
      </time>
    </li>
  );
}

const errText = (e: unknown) =>
  e instanceof ApiError && e.status === 429 ? 'You are sending messages very quickly. Please wait a moment.' : e instanceof Error ? e.message : 'Something went wrong.';

/** Chat window: a sheet over a dimmed page (bottom sheet on phones, side panel on desktop). */
export default function ChatPanel({ onClose }: { onClose: () => void }) {
  const user = useAuthStore((s) => s.user);
  const panel = useRef<HTMLDivElement>(null);
  const list = useRef<HTMLOListElement>(null);
  useOverlay(true, onClose, panel);

  const [ref, setRef] = useState<ChatRef | null>(() => loadChatRef());
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [status, setStatus] = useState<'open' | 'closed'>('open');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const [text, setText] = useState('');
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');

  const lastId = messages.length ? messages[messages.length - 1].id : 0;

  // Find the conversation: the stored one, or (signed in) the account's.
  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const c = ref ? await api.getChat(ref.id, { read: true, token: ref.token }) : user ? await api.myChat() : null;
        if (!alive) return;
        if (c) {
          setMessages(c.messages);
          setStatus(c.status);
          if (!ref) {
            const r = { id: c.id };
            saveChatRef(r);
            setRef(r);
          }
        }
      } catch {
        // Stored chat no longer reachable (e.g. signed out): start fresh.
        if (alive && ref) {
          saveChatRef(null);
          setRef(null);
        }
      } finally {
        if (alive) setLoading(false);
      }
    };
    void load();
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const poll = useCallback(() => {
    if (!ref) return;
    api.getChat(ref.id, { after: lastId, read: true, token: ref.token }).then(
      (c) => {
        setStatus(c.status);
        if (c.messages.length) setMessages((prev) => [...prev, ...c.messages.filter((m) => !prev.some((p) => p.id === m.id))]);
      },
      () => undefined,
    );
  }, [ref, lastId]);
  usePolling(poll, POLL_MS, !!ref && !loading);

  useEffect(() => {
    list.current?.lastElementChild?.scrollIntoView({ block: 'end' });
  }, [messages.length]);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const body = text.trim();
    if (!body || sending) return;
    setSending(true);
    setError(null);
    try {
      if (!ref) {
        const c = await api.startChat(user ? { message: body } : { name: name.trim(), email: email.trim(), message: body });
        const r = { id: c.id, token: c.token };
        saveChatRef(r);
        setRef(r);
        setMessages(c.messages);
        setStatus(c.status);
      } else {
        const m = await api.sendChat(ref.id, body, ref.token);
        setMessages((prev) => [...prev, m]);
        setStatus('open');
      }
      setText('');
    } catch (err) {
      setError(errText(err));
    } finally {
      setSending(false);
    }
  };

  const starting = !ref && !loading;
  return (
    <>
      <button type="button" className="backdrop" aria-label="Close chat" tabIndex={-1} onClick={onClose} />
      <div ref={panel} className="sheet sheet--bottom chat-sheet" role="dialog" aria-modal="true" aria-labelledby="chat-title">
        <div className="sheet__head chat-head">
          <div>
            <p className="sheet__title" id="chat-title">
              Live chat
            </p>
            <p className="chat-head__sub">
              <span className="chat-dot" aria-hidden="true" /> We usually reply within a few minutes, 9am–6pm UK time.
            </p>
          </div>
          <button type="button" className="sheet__close" aria-label="Close chat" onClick={onClose}>
            ✕
          </button>
        </div>

        <div className="sheet__body chat-body">
          {loading ? (
            <div className="skeleton" style={{ height: 120 }} />
          ) : (
            <>
              {starting && <p className="chat-intro">Ask about a scent, sizes, delivery or your order. A real person from the team will answer here{user ? '' : ' and by email'}.</p>}
              <ol ref={list} className="chat-list" aria-live="polite" aria-relevant="additions">
                {messages.map((m) => (
                  <Bubble key={m.id} m={m} />
                ))}
              </ol>
              {status === 'closed' && messages.length > 0 && <p className="chat-closed">This conversation was closed. Write again to reopen it.</p>}
            </>
          )}
        </div>

        <form className="sheet__foot chat-form" onSubmit={submit}>
          {starting && !user && (
            <div className="chat-form__who">
              <label className="field">
                Your name
                <input className="input" value={name} onChange={(e) => setName(e.target.value)} required maxLength={60} autoComplete="name" />
              </label>
              <label className="field">
                Email
                <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required maxLength={254} autoComplete="email" />
              </label>
            </div>
          )}
          {error && (
            <p className="chat-error" role="alert">
              {error}
            </p>
          )}
          <div className="chat-form__row">
            <label htmlFor="chat-input" className="visually-hidden">
              Your message
            </label>
            <textarea
              id="chat-input"
              className="textarea chat-form__input"
              rows={2}
              maxLength={2000}
              value={text}
              data-autofocus
              placeholder={starting ? 'How can we help?' : 'Write a message…'}
              onChange={(e) => setText(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  e.currentTarget.form?.requestSubmit();
                }
              }}
            />
            <button type="submit" className="btn btn--primary chat-form__send" disabled={sending || !text.trim()}>
              {sending ? 'Sending…' : 'Send'}
            </button>
          </div>
        </form>
      </div>
    </>
  );
}
