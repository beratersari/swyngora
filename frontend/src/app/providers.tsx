import type { ReactNode } from 'react';
import { Provider as ReduxProvider } from 'react-redux';
import { ConfigProvider } from 'antd';
import { BrowserRouter } from 'react-router-dom';
import { ThemeProvider } from 'styled-components';
import { store } from '@/libs/api';
import { antdTheme } from '@/styles/antdTheme';
import { appTheme } from '@/styles/theme';
import { GlobalStyle } from '@/styles/GlobalStyle';

type ProvidersProps = {
  children: ReactNode;
};

export function Providers({ children }: ProvidersProps) {
  return (
    <ReduxProvider store={store}>
      <ThemeProvider theme={appTheme}>
        <GlobalStyle />
        <ConfigProvider theme={antdTheme}>
          <BrowserRouter>{children}</BrowserRouter>
        </ConfigProvider>
      </ThemeProvider>
    </ReduxProvider>
  );
}
