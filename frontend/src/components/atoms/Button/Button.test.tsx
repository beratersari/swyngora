import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { Button } from './Button';

describe('Button', () => {
  it('renders children', () => {
    renderWithTheme(<Button>Save</Button>);
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
  });

  it('shows skeleton when isLoading', () => {
    const { container } = renderWithTheme(<Button isLoading>Save</Button>);
    expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
    expect(container.querySelector('.ant-skeleton-button')).toBeTruthy();
  });

  it('disables when pending and disabled omitted', () => {
    renderWithTheme(<Button pending>Go</Button>);
    expect(screen.getByRole('button', { name: /go/i })).toBeDisabled();
  });

  it('respects explicit disabled=false while pending', () => {
    renderWithTheme(
      <Button pending disabled={false}>
        Go
      </Button>,
    );
    expect(screen.getByRole('button', { name: /go/i })).not.toBeDisabled();
  });

  it('forwards click when enabled', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    renderWithTheme(<Button onClick={onClick}>Click</Button>);
    await user.click(screen.getByRole('button', { name: 'Click' }));
    expect(onClick).toHaveBeenCalledOnce();
  });
});
