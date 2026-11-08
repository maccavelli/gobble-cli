package provider

import (
	"context"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// Cache is what a run's SDK calls keep in memory, in place of the
// process's: the listings, model facts and probes of the provider, its
// listings and its model lookups (go-llmprovider-sdk catalog.Cache). Its
// owner ends it with Close, which stops the background work those calls
// started and waits for it (0046-MADR D7). It holds the SDK's cache so that
// no package but this one imports the SDK (0046-PLAN deviation 5). The zero
// value is ready to use; a Cache must not be copied after its first use.
type Cache struct{ c catalog.Cache }

// Close ends c: no call given it starts background work after Close begins,
// the work in flight is cancelled, and Close returns once that work has
// ended, or ctx.Err() if ctx ends first. A Close that returned ctx.Err() can
// be called again to wait longer (go-llmprovider-sdk catalog.Cache.Close).
// After it, a listing given c is refused, and a model lookup answers from
// what c holds.
func (c *Cache) Close(ctx context.Context) error { return c.c.Close(ctx) }

// option is c as the SDK's option.
func (c *Cache) option() llmprovider.Option { return catalog.WithCache(&c.c) }
