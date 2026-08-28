import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { SignalsRuleForm } from './SignalsRuleForm';

describe('SignalsRuleForm', () => {
  it('submits selected conditions and match mode', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <SignalsRuleForm intervals={['1h', '4h']} defaultInterval="4h" onSubmit={onSubmit} />,
    );
    await user.click(screen.getByRole('checkbox', { name: /volume increase|hacim artışı/i }));
    await user.click(screen.getByRole('button', { name: /create rule|kural oluştur/i }));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        conditions: ['rsi', 'volume_increase'],
        matchMode: 'all',
        interval: '4h',
        rsiCondition: 'below',
        rsiThreshold: 40,
        volumeLookback: 20,
        volumeMinRatio: 2,
      }),
    );
    expect(onSubmit.mock.calls[0]?.[0]).not.toHaveProperty('type');
  });

  it('saves edits to an existing rule', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <SignalsRuleForm
        intervals={['4h']}
        defaultInterval="4h"
        initialRule={{
          id: 'r1',
          type: 'rsi',
          conditions: ['rsi'],
          matchMode: 'all',
          interval: '4h',
          enabled: true,
          rsiPeriod: 14,
          rsiCondition: 'below',
          rsiThreshold: 40,
        }}
        onSubmit={onSubmit}
        onCancel={() => undefined}
      />,
    );
    expect(screen.getByRole('button', { name: /save|kaydet/i })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /save|kaydet/i }));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        conditions: ['rsi'],
        matchMode: 'all',
        rsiThreshold: 40,
      }),
    );
  });

  it('disables create when no condition is selected', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SignalsRuleForm intervals={['4h']} defaultInterval="4h" onSubmit={vi.fn()} />,
    );
    await user.click(screen.getByRole('checkbox', { name: /rsi threshold|rsi eşiği/i }));
    expect(screen.getByRole('button', { name: /create rule|kural oluştur/i })).toBeDisabled();
  });
});
