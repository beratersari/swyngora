import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { useGetFxRatesQuery, useListExchangesQuery } from '@/libs/api';
import {
  convertAmount,
  formatConvertedCompact,
  formatConvertedPrice,
  loadDisplayCurrency,
  marketCapQuote,
  saveDisplayCurrency,
  venueQuote,
  type DisplayCurrency,
  type FxRatesMap,
} from '@/libs/utils';

export type DisplayCurrencyApi = {
  currency: DisplayCurrency;
  setCurrency: (next: DisplayCurrency) => void;
  rates: FxRatesMap;
  venueQuotes: Record<string, string>;
  marketCapQuotes: Record<string, string>;
  aliases: Record<string, string>;
  asOf?: string;
  stale: boolean;
  ready: boolean;
  convert: (value: string | number | null | undefined, nativeQuote: string) => number | null;
  formatPrice: (value: string | number | null | undefined, nativeQuote: string) => string;
  formatCompact: (value: string | number | null | undefined, nativeQuote: string) => string;
  nativeQuote: (exchange?: string | null) => string;
  mcapQuote: (exchange?: string | null) => string;
};

const DisplayCurrencyContext = createContext<DisplayCurrencyApi | null>(null);

export function DisplayCurrencyProvider({ children }: { children: ReactNode }) {
  const [currency, setCurrencyState] = useState<DisplayCurrency>(loadDisplayCurrency);
  const fx = useGetFxRatesQuery(undefined, { pollingInterval: 15 * 60_000, refetchOnFocus: true });
  const exchanges = useListExchangesQuery();
  const rates = (fx.data?.rates ?? {}) as FxRatesMap;
  const venueQuotes = fx.data?.venueQuotes ?? exchanges.data?.venueQuotes ?? {};
  const marketCapQuotes = fx.data?.marketCapQuotes ?? exchanges.data?.marketCapQuotes ?? {};
  const aliases = fx.data?.aliases ?? {};

  const setCurrency = useCallback((next: DisplayCurrency) => {
    setCurrencyState(next);
    saveDisplayCurrency(next);
  }, []);

  const value = useMemo<DisplayCurrencyApi>(
    () => ({
      currency,
      setCurrency,
      rates,
      venueQuotes,
      marketCapQuotes,
      aliases,
      asOf: fx.data?.asOf,
      stale: Boolean(fx.data?.stale),
      ready: Boolean(fx.data?.rates) || fx.isError,
      convert: (amount, nativeQuote) => {
        if (currency === 'native') {
          const n = typeof amount === 'number' ? amount : Number(amount);
          return Number.isFinite(n) ? n : null;
        }
        return convertAmount(amount, nativeQuote, currency, rates, aliases);
      },
      formatPrice: (amount, nativeQuote) =>
        formatConvertedPrice(amount, nativeQuote, currency, rates, aliases),
      formatCompact: (amount, nativeQuote) =>
        formatConvertedCompact(amount, nativeQuote, currency, rates, aliases),
      nativeQuote: (exchange) => venueQuote(exchange, venueQuotes),
      mcapQuote: (exchange) => marketCapQuote(exchange, marketCapQuotes),
    }),
    [
      currency,
      rates,
      venueQuotes,
      marketCapQuotes,
      aliases,
      setCurrency,
      fx.data?.asOf,
      fx.data?.stale,
      fx.data?.rates,
      fx.isError,
    ],
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
      venueQuotes: {},
      marketCapQuotes: {},
      aliases: {},
      stale: false,
      ready: false,
      convert: (amount) => {
        const n = typeof amount === 'number' ? amount : Number(amount);
        return Number.isFinite(n) ? n : null;
      },
      formatPrice: (amount, nativeQuote) => formatConvertedPrice(amount, nativeQuote, 'native', {}, {}),
      formatCompact: (amount, nativeQuote) =>
        formatConvertedCompact(amount, nativeQuote, 'native', {}, {}),
      nativeQuote: (exchange) => venueQuote(exchange),
      mcapQuote: (exchange) => marketCapQuote(exchange),
    };
  }
  return ctx;
}
