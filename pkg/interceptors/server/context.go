package server

import "context"

// optionsKey is used as the key for associating information
// with a context.Context.
type optionsKey struct{}

// WithOptions associates a set of server.Options with
// the returned context.
func WithOptions(ctx context.Context, opt Options) context.Context {
	return context.WithValue(ctx, optionsKey{}, &opt)
}

// GetOptions retrieves webhook.Options associated with the
// given context via WithOptions (above).
func GetOptions(ctx context.Context) *Options {
	v := ctx.Value(optionsKey{})
	if v == nil {
		return nil
	}
	return v.(*Options)
}
