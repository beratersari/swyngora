import { describe, expect, it } from 'vitest';
import {
  createdKeySessionToken,
  customSignalImportPreview,
  exportDownloadHref,
} from './SettingsPage.helpers';

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

describe('customSignalImportPreview', () => {
  it('returns empty lists when the job has no customSignals section', () => {
    expect(customSignalImportPreview(undefined)).toEqual({
      addNames: [],
      updateNames: [],
      willAdd: 0,
      willUpdate: 0,
    });
    expect(customSignalImportPreview({ sections: {} })).toEqual({
      addNames: [],
      updateNames: [],
      willAdd: 0,
      willUpdate: 0,
    });
  });

  it('splits new vs existing profile names', () => {
    const got = customSignalImportPreview({
      sections: {
        customSignals: {
          willAdd: 1,
          willUpdate: 2,
          addNames: ['Trend mix'],
          updateNames: ['Hunt mix', 'Default'],
        },
      },
    });
    expect(got).toEqual({
      addNames: ['Trend mix'],
      updateNames: ['Hunt mix', 'Default'],
      willAdd: 1,
      willUpdate: 2,
    });
  });
});

describe('createdKeySessionToken', () => {
  const trade = 'swy_tradelock00000000000000000000000000000';
  const read = 'swy_readlock000000000000000000000000000000';

  it('does not bind a read key', () => {
    expect(createdKeySessionToken(read, 'read')).toBeNull();
    expect(createdKeySessionToken(read, undefined)).toBeNull();
  });

  it('binds a trade key', () => {
    expect(createdKeySessionToken(trade, 'trade')).toBe(trade);
    expect(createdKeySessionToken(` ${trade} `, 'TRADE')).toBe(trade);
  });

  it('ignores an empty secret', () => {
    expect(createdKeySessionToken('', 'trade')).toBeNull();
    expect(createdKeySessionToken(null, 'trade')).toBeNull();
  });
});
