import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { SignalsRuleForm } from './SignalsRuleForm';

describe('SignalsRuleForm', () => {
  it('submits an RSI rule', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <SignalsRuleForm intervals={['1h', '4h']} defaultInterval="4h" onSubmit={onSubmit} />,
    );
    await user.click(screen.getByRole('button', { name: /create rule|kural oluştur/i }));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'rsi',
        interval: '4h',
        rsiCondition: 'below',
        rsiThreshold: 40,
      }),
    );
  });
});
