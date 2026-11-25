package provider

import (
	"fmt"
	"io"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// ModelData is a parsed models.dev document, immutable once parsed: what a
// listing ranks by and reads its menu's facts from, handed in through
// Options.ModelData (0071-MADR P3; 0072-MADR D21). go-llmprovider-sdk
// fetches no document from v1.33.0: a listing given none ranks by the SDK's
// curated order and name rules (its 0093-MADR D3). A ModelData is safe for
// concurrent use.
type ModelData struct {
	meta catalog.Metadata
}

// ParseModelData reads a models.dev document from r with the SDK's parser,
// which refuses a document over 32 MiB or one that is not a document
// (catalog.ParseMetadata).
func ParseModelData(r io.Reader) (*ModelData, error) {
	meta, err := catalog.ParseMetadata(r)
	if err != nil {
		return nil, fmt.Errorf("provider: model data: %w", err)
	}
	return &ModelData{meta: meta}, nil
}

// options are the SDK options that hand d to a listing; nil hands none.
func (d *ModelData) options() []llmprovider.Option {
	if d == nil {
		return nil
	}
	return []llmprovider.Option{catalog.WithMetadata(d.meta)}
}
