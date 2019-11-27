package v1alpha1

import (
	"context"
)

// SetDefaults sets the defaults on the object.
func (el *EventListener) SetDefaults(ctx context.Context) {
	if IsUpgradeViaDefaulting(ctx) {
		// Most likely the EventListener passed here is already running
		for i := range el.Spec.Triggers {
			t := &el.Spec.Triggers[i]
			if t.DeprecatedBinding != nil {
				if len(t.Bindings) > 0 {
					// Do nothing since it will be a Validaiton Error.
				} else {
					// Set the binding to bindings
					t.Bindings = append(t.Bindings, &EventListenerBinding{
						Name: t.DeprecatedBinding.Name,
					})
					t.DeprecatedBinding = nil
				}
			}
		}
	}
}
