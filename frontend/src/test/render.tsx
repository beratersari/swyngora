import type { ReactElement, ReactNode } from 'react';
import { render, type RenderOptions } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { I18nextProvider } from 'react-i18next';
import { Provider as ReduxProvider } from 'react-redux';
import { MemoryRouter, type MemoryRouterProps } from 'react-router-dom';
import { ThemeProvider } from 'styled-components';
import { store } from '@/libs/api';
import { getAntdLocale, i18n } from '@/libs/i18n';
import { antdTheme } from '@/styles/antdTheme';
import { appTheme } from '@/styles/theme';

function ThemeI18nProviders({ children }: { children: ReactNode }) {
  return (
    <I18nextProvider i18n={i18n}>
      <ThemeProvider theme={appTheme}>{children}</ThemeProvider>
    </I18nextProvider>
  );
}

/** RTL render with i18n + styled-components ThemeProvider (presentational atoms). */
export function renderWithTheme(ui: ReactElement, options?: Omit<RenderOptions, 'wrapper'>) {
  return render(ui, { wrapper: ThemeI18nProviders, ...options });
}

export type RenderWithProvidersOptions = Omit<RenderOptions, 'wrapper'> & {
  /** Initial MemoryRouter entries (default: `['/']`). */
  routerEntries?: MemoryRouterProps['initialEntries'];
};

/**
 * Full app stack for pages/routes: Redux + theme + Ant locale + MemoryRouter + i18n.
 * Prefer this for anything that needs RTK Query or react-router.
 */
export function renderWithProviders(
  ui: ReactElement,
  { routerEntries = ['/'], ...options }: RenderWithProvidersOptions = {},
) {
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <ReduxProvider store={store}>
        <I18nextProvider i18n={i18n}>
          <ThemeProvider theme={appTheme}>
            <ConfigProvider theme={antdTheme} locale={getAntdLocale(i18n.language)}>
              <MemoryRouter initialEntries={routerEntries}>{children}</MemoryRouter>
            </ConfigProvider>
          </ThemeProvider>
        </I18nextProvider>
      </ReduxProvider>
    );
  }
  return render(ui, { wrapper: Wrapper, ...options });
}
