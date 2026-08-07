import { describe, expect, it } from 'vitest';
import { renderWithTheme } from '@/test/render';
import { PageHeader } from './PageHeader';

describe('PageHeader', () => {
  it('renders title as heading and optional eyebrow/subtitle/extra', () => {
    const { getByRole, getByText } = renderWithTheme(
      <PageHeader
        eyebrow="Spot"
        title="Markets"
        subtitle="Live books"
        extra={<button type="button">Cols</button>}
      />,
    );
    expect(getByRole('heading', { name: 'Markets' })).toBeInTheDocument();
    expect(getByText('Spot')).toBeInTheDocument();
    expect(getByText('Live books')).toBeInTheDocument();
    expect(getByRole('button', { name: 'Cols' })).toBeInTheDocument();
  });
});
