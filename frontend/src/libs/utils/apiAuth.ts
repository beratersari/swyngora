/** User-issued API key (`swy_…`) the browser sends on REST and WS. Never the process master token. */
export const BROWSER_API_TOKEN_KEY = 'swyngora.apiAuthToken';

export function getBrowserApiToken(): string {
  if (typeof localStorage === 'undefined') return '';
  try {
    return (localStorage.getItem(BROWSER_API_TOKEN_KEY) || '').trim();
  } catch {
    return '';
  }
}

export function setBrowserApiToken(token: string): void {
  if (typeof localStorage === 'undefined') return;
  const t = token.trim();
  try {
    if (!t) localStorage.removeItem(BROWSER_API_TOKEN_KEY);
    else localStorage.setItem(BROWSER_API_TOKEN_KEY, t);
  } catch {
    /* private mode / quota */
  }
}
