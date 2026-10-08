// Package facades is the facade-style accessor for goravel-myanmar-payments.
package facades

import (
	payments "github.com/laranex/goravel-myanmar-payments/v4"
)

// MyanmarPayments returns the application's *payments.Manager. It panics when
// &payments.ServiceProvider{} is not registered, like a missing facade in Goravel.
// Gateway credentials are checked when a gateway is first requested, e.g.
// MyanmarPayments().KbzPay() returns a *myanmarpayments.ConfigurationError.
func MyanmarPayments() *payments.Manager {
	manager, err := payments.Registered()
	if err != nil {
		panic(err)
	}

	return manager
}
