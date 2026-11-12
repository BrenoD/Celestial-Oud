/**
 * Guest order access: the API returns a secret token when an order is created.
 * It is kept for this browser tab only (sessionStorage) and also arrives in the
 * confirmation email link, so the order page works after a reload or later.
 */
const key = (id: string) => `co.order.${id.toUpperCase()}`;

export function rememberOrderToken(id: string, token: string): void {
  try {
    sessionStorage.setItem(key(id), token);
  } catch {
    /* storage unavailable: the email link still works */
  }
}

export function orderToken(id: string): string | null {
  try {
    return sessionStorage.getItem(key(id));
  } catch {
    return null;
  }
}
