import { lazy, Suspense, useCallback, useEffect, useState } from 'react';
import { api } from '@/api';
import { useAuthStore } from '@/store/auth';
import { getConsent } from '@/lib/consent';
import { usePolling } from '@/lib/usePolling';
import { loadChatRef, OPEN_CHAT_EVENT } from './chatStore';
import './Chat.css';

// The chat window is its own chunk, fetched on first click: the storefront
// pays only for this button (keeps Lighthouse at 100).
const ChatPanel = lazy(() => import('./ChatPanel'));

function ChatIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <path d="M4 5h16v11H9l-5 4z" strokeLinejoin="round" />
      <path d="M8 9.5h8M8 12.5h5" strokeLinecap="round" />
    </svg>
  );
}

export function ChatLauncher() {
  const [open, setOpen] = useState(false);
  const [unread, setUnread] = useState(0);
  // Out of the way while the cookie banner asks for a choice.
  const [consented, setConsented] = useState(() => getConsent() !== null);
  const user = useAuthStore((s) => s.user);

  useEffect(() => {
    const onConsent = () => setConsented(true);
    const onOpen = () => setOpen(true);
    window.addEventListener('co:consent', onConsent);
    window.addEventListener(OPEN_CHAT_EVENT, onOpen);
    return () => {
      window.removeEventListener('co:consent', onConsent);
      window.removeEventListener(OPEN_CHAT_EVENT, onOpen);
    };
  }, []);

  // While closed, check now and then for replies (only if a chat exists).
  const ref = loadChatRef();
  const check = useCallback(() => {
    const r = loadChatRef();
    if (!r) return;
    api.getChat(r.id, { after: Number.MAX_SAFE_INTEGER, token: r.token }).then(
      (c) => setUnread(c.unread),
      () => undefined,
    );
  }, []);
  usePolling(check, 30_000, !open && !!ref && (!!ref.token || !!user));

  if (!consented && !open) return null;
  return (
    <>
      {!open && (
        <button type="button" className="chat-fab" onClick={() => setOpen(true)} aria-haspopup="dialog" aria-label={unread ? `Live chat, ${unread} new ${unread === 1 ? 'reply' : 'replies'}` : 'Live chat'}>
          <ChatIcon />
          <span className="chat-fab__label">Chat with us</span>
          {unread > 0 && (
            <span className="chat-fab__badge" aria-hidden="true">
              {unread > 9 ? '9+' : unread}
            </span>
          )}
        </button>
      )}
      {open && (
        <Suspense fallback={null}>
          <ChatPanel
            onClose={() => {
              setOpen(false);
              setUnread(0);
            }}
          />
        </Suspense>
      )}
    </>
  );
}
