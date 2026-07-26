import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { Provider as ReduxProvider } from 'react-redux';
import { ConfigProvider } from 'antd';
import { BrowserRouter } from 'react-router-dom';
import { ThemeProvider } from 'styled-components';
import { useTranslation } from 'react-i18next';
import { store } from '@/libs/api';
import { getAntdLocale } from '@/libs/i18n';
import { antdTheme } from '@/styles/antdTheme';
import { appTheme } from '@/styles/theme';
import { GlobalStyle } from '@/styles/GlobalStyle';

type ProvidersProps = {
  children: ReactNode;
};

function AntdLocaleBridge({ children }: { children: ReactNode }) {
  const { i18n } = useTranslation();
  const [lng, setLng] = useState(i18n.language);

  useEffect(() => {
    const onChange = (next: string) => setLng(next);
    i18n.on('languageChanged', onChange);
    return () => {
      i18n.off('languageChanged', onChange);
    };
  }, [i18n]);

  const locale = useMemo(() => getAntdLocale(lng), [lng]);

  return (
    <ConfigProvider theme={antdTheme} locale={locale}>
      {children}
    </ConfigProvider>
  );
}

export function Providers({ children }: ProvidersProps) {
  return (
    <ReduxProvider store={store}>
      <ThemeProvider theme={appTheme}>
        <GlobalStyle />
        <AntdLocaleBridge>
          <BrowserRouter>{children}</BrowserRouter>
        </AntdLocaleBridge>
      </ThemeProvider>
    </ReduxProvider>
  );
}
