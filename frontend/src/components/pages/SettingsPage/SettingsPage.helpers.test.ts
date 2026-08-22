import { describe, expect, it } from 'vitest';
import { exportDownloadHref } from './SettingsPage.helpers';

describe('exportDownloadHref', () => {
  it('returns null when missing', () => {
    expect(exportDownloadHref(undefined)).toBeNull();
    expect(exportDownloadHref('')).toBeNull();
  });

  it('keeps absolute URLs', () => {
    expect(exportDownloadHref('https://api.example/export/1')).toBe('https://api.example/export/1');
  });

  it('prefixes a relative path', () => {
    const href = exportDownloadHref('/api/v1/export/abc/download');
    expect(href).toContain('/api/v1/export/abc/download');
  });
});
