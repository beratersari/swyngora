export type PortfolioCorrelationProps = {
  portfolioId?: string;
  disabled?: boolean;
};

export type CorrelationKind = 'empty' | 'one' | 'together' | 'hedge' | 'inverse' | 'mixed' | 'incomplete';
