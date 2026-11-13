import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import { useAuthStore } from '@/store/auth';
import { usePolling } from '@/lib/usePolling';
import { adminApi } from '../api';
import { canWrite } from '../rbac';
import type { AdminChat } from '../types';
import { errMsg, LoadState, useToast } from '../components/ui';

const LIST_POLL = 5_000;
const THREAD_POLL = 3_000;

const when = (iso: string) => {
  const d = new Date(iso);
  const today = new Date().toDateString() === d.toDateString();
  return today ? d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' }) : d.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' });
};

type Msg = NonNullable<AdminChat['messages']>[number];

function Thread({ chatId, writable, onChanged }: { chatId: number; writable: boolean; onChanged: () => void }) {
  const toast = useToast();
  const [chat, setChat] = useState<AdminChat | null>(null);
  const [msgs, setMsgs] = useState<Msg[]>([]);
  const [text, setText] = useState('');
  const [busy, setBusy] = useState(false);
  const end = useRef<HTMLLIElement>(null);
  const lastId = msgs.length ? msgs[msgs.length - 1].id : 0;

  useEffect(() => {
    let alive = true;
    adminApi.chats.get(chatId).then(
      (c) => {
        if (!alive) return;
        setChat(c);
        setMsgs(c.messages ?? []);
        onChanged();
      },
      (e) => toast(errMsg(e), 'err'),
    );
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chatId]);

  const poll = useCallback(() => {
    adminApi.chats.get(chatId, lastId).then(
      (c) => {
        setChat((prev) => (prev ? { ...prev, status: c.status } : c));
        if (c.messages?.length) setMsgs((prev) => [...prev, ...c.messages!.filter((m) => !prev.some((p) => p.id === m.id))]);
      },
      () => undefined,
    );
  }, [chatId, lastId]);
  usePolling(poll, THREAD_POLL, !!chat);

  useEffect(() => {
    end.current?.scrollIntoView({ block: 'end' });
  }, [msgs.length]);

  const send = async (e: FormEvent) => {
    e.preventDefault();
    const body = text.trim();
    if (!body) return;
    setBusy(true);
    try {
      const m = await adminApi.chats.reply(chatId, body);
      setMsgs((prev) => [...prev, m]);
      setText('');
      setChat((c) => (c ? { ...c, status: 'open' } : c));
      onChanged();
    } catch (err) {
      toast(errMsg(err), 'err');
    } finally {
      setBusy(false);
    }
  };

  const setStatus = async (status: 'open' | 'closed') => {
    try {
      await adminApi.chats.setStatus(chatId, status);
      setChat((c) => (c ? { ...c, status } : c));
      toast(status === 'closed' ? 'Conversation closed.' : 'Conversation reopened.');
      onChanged();
    } catch (err) {
      toast(errMsg(err), 'err');
    }
  };

  if (!chat) return <div className="skeleton" style={{ height: 320 }} />;
  return (
    <section className="achat-thread" aria-label={`Conversation with ${chat.customerName}`}>
      <header className="achat-thread__head">
        <div>
          <b>{chat.customerName}</b>
          <span className="form-hint">
            {chat.customerEmail} · {chat.userId ? 'Customer account' : 'Guest'} · started {when(chat.createdAt)}
          </span>
        </div>
        {writable &&
          (chat.status === 'open' ? (
            <button type="button" className="btn btn--outline btn--xs" onClick={() => setStatus('closed')}>
              Close
            </button>
          ) : (
            <button type="button" className="btn btn--outline btn--xs" onClick={() => setStatus('open')}>
              Reopen
            </button>
          ))}
      </header>
      <ol className="achat-msgs" aria-live="polite" aria-relevant="additions">
        {msgs.map((m) => (
          <li key={m.id} className={`achat-msg achat-msg--${m.sender}`}>
            <p>{m.body}</p>
            <time dateTime={m.createdAt}>
              {m.sender === 'agent' ? 'Team · ' : ''}
              {when(m.createdAt)}
            </time>
          </li>
        ))}
        <li ref={end} aria-hidden="true" />
      </ol>
      {chat.status === 'closed' && <p className="form-hint" style={{ textAlign: 'center' }}>Closed. Replying reopens it.</p>}
      {writable && (
        <form className="achat-reply" onSubmit={send}>
          <label htmlFor="achat-input" className="visually-hidden">
            Reply
          </label>
          <textarea
            id="achat-input"
            className="textarea"
            rows={2}
            maxLength={2000}
            value={text}
            placeholder="Write a reply… (Enter to send, Shift+Enter for a new line)"
            onChange={(e) => setText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                e.currentTarget.form?.requestSubmit();
              }
            }}
          />
          <button type="submit" className="btn btn--primary btn--sm" disabled={busy || !text.trim()}>
            {busy ? 'Sending…' : 'Send'}
          </button>
        </form>
      )}
    </section>
  );
}

/** Team inbox: live list with shared unread counts, conversation on the right. */
export default function Chats() {
  const writable = canWrite(useAuthStore((s) => s.user?.role), 'chats');
  const [status, setStatus] = useState<'open' | 'closed' | ''>('open');
  const [chats, setChats] = useState<AdminChat[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [active, setActive] = useState<number | null>(null);

  const load = useCallback(() => {
    adminApi.chats.list(status || undefined).then(
      (c) => {
        setChats(c);
        setError(null);
      },
      (e) => setError(errMsg(e)),
    );
  }, [status]);
  useEffect(load, [load]);
  usePolling(load, LIST_POLL);

  return (
    <>
      <div className="adm-toolbar">
        <label className="field" htmlFor="chat-status">
          Show
          <select id="chat-status" className="select" value={status} onChange={(e) => setStatus(e.target.value as typeof status)}>
            <option value="open">Open</option>
            <option value="closed">Closed</option>
            <option value="">All</option>
          </select>
        </label>
        <span className="form-hint">Updates every few seconds. Opening a conversation marks it read for the whole team.</span>
      </div>
      <LoadState loading={!chats && !error} error={error} />
      {chats && (
        <div className={`achat${active ? ' has-active' : ''}`}>
          <ul className="achat-list">
            {chats.map((c) => (
              <li key={c.id}>
                <button type="button" className={`achat-item${c.id === active ? ' is-active' : ''}${c.unread ? ' is-unread' : ''}`} onClick={() => setActive(c.id)} aria-current={c.id === active}>
                  <span className="achat-item__top">
                    <b>{c.customerName}</b>
                    <span className="form-hint">{when(c.lastMessageAt)}</span>
                  </span>
                  <span className="achat-item__last">{c.lastMessage ?? '—'}</span>
                  {c.unread > 0 && (
                    <span className="nav-badge achat-item__badge" aria-label={`${c.unread} unread`}>
                      {c.unread}
                    </span>
                  )}
                </button>
              </li>
            ))}
            {!chats.length && <li className="form-hint" style={{ padding: 16 }}>No {status || ''} conversations.</li>}
          </ul>
          {active ? (
            <div className="achat-pane">
              <button type="button" className="btn btn--ghost btn--xs achat-back" onClick={() => setActive(null)}>
                ← All conversations
              </button>
              <Thread key={active} chatId={active} writable={writable} onChanged={load} />
            </div>
          ) : (
            <p className="form-hint achat-empty">Choose a conversation.</p>
          )}
        </div>
      )}
    </>
  );
}
