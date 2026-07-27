export const PumpsScreens = {
  Scan: 'PumpsScan',
  Detail: 'CoinDetail',
} as const;

export type PumpsStackParamList = {
  [PumpsScreens.Scan]: undefined;
  [PumpsScreens.Detail]: {
    exchange: string;
    symbol: string;
  };
};
