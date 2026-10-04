package gobble_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/maccavelli/gobble-cli/gobble"
)

func ExampleNew() {
	ctx := context.Background()
	rt, err := gobble.New(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = rt.ServeACP(ctx, nil, nil)
	fmt.Println(errors.Is(err, errors.ErrUnsupported))
	// Output:
	// true
}
