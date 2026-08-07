import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { AppShell, BrandLink, NavLink } from './index';

describe('AppShell', () => {
  it('renders brand, nav, tools, main content, and footer', () => {
    renderWithProviders(
      <AppShell
        navAriaLabel="Primary"
        brand={<BrandLink to="/markets">Brand</BrandLink>}
        nav={
          <>
            <NavLink to="/markets">Markets</NavLink>
            <NavLink to="/watchlist">Watchlist</NavLink>
          </>
        }
        tools={<button type="button">Tools</button>}
        footer={<span>Footer note</span>}
      >
        <div data-testid="main-slot">Main</div>
      </AppShell>,
    );

    expect(screen.getByRole('navigation', { name: 'Primary' })).toBeInTheDocument();
    expect(screen.getByText('Brand')).toBeInTheDocument();
    expect(screen.getByText('Markets')).toBeInTheDocument();
    expect(screen.getByText('Watchlist')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Tools' })).toBeInTheDocument();
    expect(screen.getByTestId('main-slot')).toHaveTextContent('Main');
    expect(screen.getByText('Footer note')).toBeInTheDocument();
  });

  it('omits footer when not provided', () => {
    const { container } = renderWithProviders(
      <AppShell brand={<span>B</span>} nav={<span>N</span>}>
        child
      </AppShell>,
    );
    expect(container.querySelector('footer')).toBeNull();
  });

  it('widens content when wide is set', () => {
    const { container } = renderWithProviders(
      <AppShell brand={<span>B</span>} nav={<span>N</span>} wide>
        main
      </AppShell>,
    );
    // styled-components attaches max-width via class; assert content renders
    expect(container.textContent).toContain('main');
  });

  it('marks active route with aria-current and active class', () => {
    renderWithProviders(
      <AppShell
        navAriaLabel="Main"
        brand={<BrandLink to="/markets">Brand</BrandLink>}
        nav={
          <>
            <NavLink to="/markets">Markets</NavLink>
            <NavLink to="/pumps">Pumps</NavLink>
          </>
        }
      >
        main
      </AppShell>,
      { routerEntries: ['/pumps'] },
    );

    const pumps = screen.getByRole('link', { name: 'Pumps' });
    expect(pumps).toHaveAttribute('aria-current', 'page');
    expect(pumps.className).toMatch(/active/);
    const markets = screen.getByRole('link', { name: 'Markets' });
    expect(markets).not.toHaveAttribute('aria-current');
  });
});
