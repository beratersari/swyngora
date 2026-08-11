import { describe, expect, it } from 'vitest';
import { renderWithTheme } from '@/test/render';
import { ConnectionStatus } from './ConnectionStatus';

describe('ConnectionStatus', () => {
  it('announces the label as a live status', () => {
    const { getByRole } = renderWithTheme(<ConnectionStatus status="live" label="Live" />);
    expect(getByRole('status')).toHaveTextContent('Live');
  });
});
