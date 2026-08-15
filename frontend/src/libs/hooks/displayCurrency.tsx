import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { useGetFxRatesQuery } from '@/libs/api';
import {
  convertAmount,
  formatConvertedCompact,
  formatConvertedPrice,
  loadDisplayCurrency,
  saveDisplayCurrency,
  type DisplayCurrency,
  type FxRatesMap,
} from '@/libs/utils';

export type DisplayCurrencyApi = {
  currency: DisplayCurrency;
  setCurrency: (next: DisplayCurrency) => void;
  rates: FxRatesMap;
  asOf?: string;
  stale: boolean;
  ready: boolean;
  convert: (value: string | number | null | undefined, nativeQuote: string) => number | null;
  formatPrice: (value: string | number | null | undefined, nativeQuote: string) => string;
  formatCompact: (value: string | number | null | undefined, nativeQuote: string) => string;
};

const DisplayCurrencyContext = createContext<DisplayCurrencyApi | null>(null);

export function DisplayCurrencyProvider({ children }: { children: ReactNode }) {
  const [currency, setCurrencyState] = useState<DisplayCurrency>(loadDisplayCurrency);
  const fx = useGetFxRatesQuery(undefined, { pollingInterval: 15 * 60_000, refetchOnFocus: true });
  const rates = (fx.data?.rates ?? {}) as FxRatesMap;

  const setCurrency = useCallback((next: DisplayCurrency) => {
    setCurrencyState(next);
    saveDisplayCurrency(next);
  }, []);

  const value = useMemo<DisplayCurrencyApi>(
    () => ({
      currency,
      setCurrency,
      rates,
      asOf: fx.data?.asOf,
      stale: Boolean(fx.data?.stale),
      ready: Boolean(fx.data?.rates) || fx.isError,
      convert: (amount, nativeQuote) => {
        if (currency === 'native') {
          const n = typeof amount === 'number' ? amount : Number(amount);
          return Number.isFinite(n) ? n : null;
        }
        return convertAmount(amount, nativeQuote, currency, rates);
      },
      formatPrice: (amount, nativeQuote) => formatConvertedPrice(amount, nativeQuote, currency, rates),
      formatCompact: (amount, nativeQuote) => formatConvertedCompact(amount, nativeQuote, currency, rates),
    }),
    [currency, rates, setCurrency, fx.data?.asOf, fx.data?.stale, fx.data?.rates, fx.isError],
  );

  return <DisplayCurrencyContext.Provider value={value}>{children}</DisplayCurrencyContext.Provider>;
}

export function useDisplayCurrency(): DisplayCurrencyApi {
  const ctx = useContext(DisplayCurrencyContext);
  if (!ctx) {
    return {
      currency: 'native',
      setCurrency: () => undefined,
      rates: {},
      stale: false,
      ready: false,
      convert: (amount) => {
        const n = typeof amount === 'number' ? amount : Number(amount);
        return Number.isFinite(n) ? n : null;
      },
      formatPrice: (amount, nativeQuote) => formatConvertedPrice(amount, nativeQuote, 'native', {}),
      formatCompact: (amount, nativeQuote) => formatConvertedCompact(amount, nativeQuote, 'native', {}),
    };
  }
  return ctx;
}
