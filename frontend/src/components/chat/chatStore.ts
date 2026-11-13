/** Which conversation this browser is in. Guests also keep their secret token. */
const KEY = 'co.chat';
export const OPEN_CHAT_EVENT = 'co:open-chat';

export interface ChatRef {
  id: number;
  token?: string;
}

export function loadChatRef(): ChatRef | null {
  try {
    const v = JSON.parse(localStorage.getItem(KEY) ?? 'null') as ChatRef | null;
    return v && typeof v.id === 'number' ? v : null;
  } catch {
    return null;
  }
}

export function saveChatRef(ref: ChatRef | null): void {
  try {
    if (ref) localStorage.setItem(KEY, JSON.stringify(ref));
    else localStorage.removeItem(KEY);
  } catch {
    /* storage unavailable: the chat lasts for this page view */
  }
}

/** Opens the chat from anywhere (e.g. the Help page). */
export function openChat(): void {
  window.dispatchEvent(new Event(OPEN_CHAT_EVENT));
}
