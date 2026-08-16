import { describe, expect, it } from 'vitest';
import { applyPortfolioEvent } from './applyUpdates';

describe('applyPortfolioEvent order/margin caches (finding 8)', () => {
  it('patches listPortfolioOrders and listMarginPositions on portfolio events', () => {
    const src = applyPortfolioEvent.toString();
    const touchesOrders = /listPortfolioOrders/.test(src);
    const touchesMargin = /listMarginPositions/.test(src);
    const touchesGetPortfolio = /getPortfolio/.test(src);

    expect(touchesGetPortfolio).toBe(true);

    if (!touchesOrders || !touchesMargin) {
      throw new Error(
        `CONFIRMED finding 8: applyPortfolioEvent updates getPortfolio=${touchesGetPortfolio} ` +
          `but listPortfolioOrders=${touchesOrders} listMarginPositions=${touchesMargin}`,
      );
    }
  });
});
