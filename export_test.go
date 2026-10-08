package payments

import "github.com/goravel/framework/contracts/foundation"

// resetRegistered clears the registered application between tests.
func resetRegistered() { registeredApp.Store(nil) }

// setRegistered registers app without going through ServiceProvider.Register.
func setRegistered(app foundation.Application) { registeredApp.Store(&appHolder{app: app}) }
