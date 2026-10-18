package portfolio

import (
	"fmt"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

var (
	errPriceHistoryUnset = fmt.Errorf("%w: price history is not configured", domain.ErrUpstream)
	errPriceHistoryRead  = fmt.Errorf("%w: the price history could not be read", domain.ErrUpstream)
	errNoOpenPosition    = fmt.Errorf("%w: the book has no open position to reprice", domain.ErrInvalidArgument)
	errNoReplayClose     = fmt.Errorf("%w: no open coin has a close at both ends of that window", domain.ErrInvalidArgument)
)
