// Package holders cascades public holder sources until one returns a snapshot.
package holders

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// CMCSource is CoinMarketCap (catalog id, then public slug).
// GTSource is GeckoTerminal token-info holder counts.
// EthSource is Ethplorer ERC-20 counts + top wallets.
type CMCSource interface {
	GetHolders(ctx context.Context, asset string) (*domain.AssetHolders, error)
}

type contractHolders interface {
	FromContracts(ctx context.Context, asset string, contracts []domain.AssetContract) (*domain.AssetHolders, error)
}

// Cascade tries CMC, CoinMetrics, then contract-based GeckoTerminal / Ethplorer / Routescan / Tronscan.
type Cascade struct {
	cmc       CMCSource
	metrics   CMCSource
	profile   domain.AssetProfilePort
	contracts domain.TokenContractPort
	gt        contractHolders
	eth       contractHolders
	scan      contractHolders
	tron      contractHolders
	cache     *cache.TTL[*domain.AssetHolders]
}

// Options wires holder fallbacks. CMC is required.
type Options struct {
	CMC       CMCSource
	Metrics   CMCSource
	Profile   domain.AssetProfilePort
	Contracts domain.TokenContractPort
	GeckoTerm contractHolders
	Ethplorer contractHolders
	RouteScan contractHolders
	TronScan  contractHolders
	Cache     *cache.TTL[*domain.AssetHolders]
}

// New returns a HoldersPort that walks free public sources in order.
func New(opts Options) *Cascade {
	return &Cascade{
		cmc:       opts.CMC,
		metrics:   opts.Metrics,
		profile:   opts.Profile,
		contracts: opts.Contracts,
		gt:        opts.GeckoTerm,
		eth:       opts.Ethplorer,
		scan:      opts.RouteScan,
		tron:      opts.TronScan,
		cache:     opts.Cache,
	}
}

// GetHolders tries CoinMarketCap, CoinMetrics, then contract-based public explorers.
func (c *Cascade) GetHolders(ctx context.Context, asset string) (*domain.AssetHolders, error) {
	if c == nil || c.cmc == nil {
		return nil, fmt.Errorf("%w: holders cascade not configured", domain.ErrUpstream)
	}
	key := domain.NormalizeAssetKey(asset)
	if key == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	if c.cache != nil {
		if hit, ok := c.cache.Get(key); ok && domain.HoldersUseful(hit) {
			return domain.CloneHolders(hit), nil
		}
	}

	var last error
	if snap, err := c.cmc.GetHolders(ctx, asset); domain.HoldersUseful(snap) {
		return c.store(key, snap), nil
	} else {
		last = keepHolderErr(last, err)
	}

	if c.metrics != nil {
		if snap, err := c.metrics.GetHolders(ctx, asset); domain.HoldersUseful(snap) {
			return c.store(key, snap), nil
		} else {
			last = keepHolderErr(last, err)
		}
	}

	contracts := c.lookupContracts(ctx, asset)
	if c.gt != nil && len(contracts) > 0 {
		if snap, err := c.gt.FromContracts(ctx, key, contracts); domain.HoldersUseful(snap) {
			return c.store(key, snap), nil
		} else {
			last = keepHolderErr(last, err)
		}
	}
	if c.eth != nil && len(contracts) > 0 {
		if snap, err := c.eth.FromContracts(ctx, key, contracts); domain.HoldersUseful(snap) {
			return c.store(key, snap), nil
		} else {
			last = keepHolderErr(last, err)
		}
	}
	if c.scan != nil && len(contracts) > 0 {
		if snap, err := c.scan.FromContracts(ctx, key, contracts); domain.HoldersUseful(snap) {
			return c.store(key, snap), nil
		} else {
			last = keepHolderErr(last, err)
		}
	}
	if c.tron != nil && len(contracts) > 0 {
		if snap, err := c.tron.FromContracts(ctx, key, contracts); domain.HoldersUseful(snap) {
			return c.store(key, snap), nil
		} else {
			last = keepHolderErr(last, err)
		}
	}

	if c.cache != nil {
		if stale, ok := c.cache.GetStale(key); ok && domain.HoldersUseful(stale) {
			cp := domain.CloneHolders(stale)
			cp.Stale = true
			return cp, nil
		}
	}
	if last != nil {
		return nil, last
	}
	return nil, fmt.Errorf("%w: holders for %q", domain.ErrHoldersUnpublished, key)
}

func (c *Cascade) lookupContracts(ctx context.Context, asset string) []domain.AssetContract {
	var out []domain.AssetContract
	seen := map[string]struct{}{}
	add := func(list []domain.AssetContract) {
		for _, con := range list {
			addr := strings.TrimSpace(con.Address)
			if addr == "" {
				continue
			}
			k := strings.ToLower(con.Chain) + "|" + strings.ToLower(addr)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, con)
		}
	}
	if c.profile != nil {
		if p, err := c.profile.GetAssetProfile(ctx, asset); err == nil && p != nil {
			add(p.Contracts)
		}
	}
	if c.contracts != nil {
		if got, err := c.contracts.LookupContracts(ctx, asset); err == nil {
			add(got)
		}
	}
	return out
}

// keepHolderErr prefers a miss (unpublished / unmapped / not found) over a
// later hop's upstream 4xx so the API does not report 502 when the asset
// simply has no public holder table.
func keepHolderErr(last, err error) error {
	if err == nil {
		return last
	}
	if last == nil {
		return err
	}
	if isHolderMiss(last) && isHolderUpstream(err) {
		return last
	}
	return err
}

func isHolderMiss(err error) bool {
	return errors.Is(err, domain.ErrHoldersUnpublished) ||
		errors.Is(err, domain.ErrCatalogUnmapped) ||
		errors.Is(err, domain.ErrNotFound)
}

func isHolderUpstream(err error) bool {
	return errors.Is(err, domain.ErrUpstream) || errors.Is(err, domain.ErrRateLimited)
}

func (c *Cascade) store(key string, snap *domain.AssetHolders) *domain.AssetHolders {
	if snap != nil && strings.TrimSpace(snap.Asset) == "" {
		snap.Asset = key
	}
	if c.cache != nil && snap != nil {
		c.cache.Set(key, snap)
	}
	return domain.CloneHolders(snap)
}
