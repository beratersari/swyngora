import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { ErrorBoundary } from './ErrorBoundary';

function Bomb({ fire }: { fire: boolean }) {
  if (fire) throw new Error('boom');
  return <div>ok</div>;
}

describe('ErrorBoundary', () => {
  it('renders children when healthy', () => {
    renderWithTheme(
      <ErrorBoundary>
        <Bomb fire={false} />
      </ErrorBoundary>,
    );
    expect(screen.getByText('ok')).toBeInTheDocument();
  });

  it('shows fallback UI when a child throws', () => {
    // Suppress React error boundary noise in test output
    const spy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    renderWithTheme(
      <ErrorBoundary fallbackTitle="Crashed" fallbackBody="Try again">
        <Bomb fire />
      </ErrorBoundary>,
    );
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText('Crashed')).toBeInTheDocument();
    expect(screen.getByText('Try again')).toBeInTheDocument();
    spy.mockRestore();
  });

  it('retry resets error state', async () => {
    const user = userEvent.setup();
    const spy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    let fire = true;
    function ToggleBomb() {
      if (fire) throw new Error('boom');
      return <div>recovered</div>;
    }
    renderWithTheme(
      <ErrorBoundary fallbackTitle="Crashed" fallbackBody="Try again">
        <ToggleBomb />
      </ErrorBoundary>,
    );
    fire = false;
    await user.click(screen.getByRole('button', { name: /retry/i }));
    expect(screen.getByText('recovered')).toBeInTheDocument();
    spy.mockRestore();
  });
});
