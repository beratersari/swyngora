import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { DeskEmpty } from './DeskEmpty';

describe('DeskEmpty', () => {
  it('renders title, hint, and extra', () => {
    renderWithTheme(
      <DeskEmpty title="No rows" hint="Adjust filters" extra={<button type="button">Retry</button>} />,
    );
    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.getByText('No rows')).toBeInTheDocument();
    expect(screen.getByText('Adjust filters')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
  });
});
