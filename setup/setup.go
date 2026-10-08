// Command setup is run by `./artisan package:install github.com/laranex/goravel-myanmar-payments/v4`
// (and package:uninstall). It registers the service provider in
// bootstrap/providers.go, writes config/myanmar_payments.go and appends the
// gateway variables to .env.example.
package main

import (
	"os"

	"github.com/goravel/framework/packages"
	"github.com/goravel/framework/packages/modify"
	"github.com/goravel/framework/support/path"
)

func main() {
	setup := packages.Setup(os.Args)
	moduleImport := setup.Paths().Module().Import()
	provider := "&payments.ServiceProvider{}"
	configPath := path.Config("myanmar_payments.go")
	envExample := path.Base(".env.example")

	setup.Install(
		modify.RegisterProvider(moduleImport, provider),
		modify.WhenFileNotExists(configPath,
			modify.File(configPath).Overwrite(configStub(setup.Paths().Config().Package(), setup.Paths().Facades().Import())),
		),
		modify.WhenFileExists(envExample,
			modify.WhenFileNotContains(envExample, "KBZ_PAY_APP_ID", modify.File(envExample).Append(envStub())),
		),
	).Uninstall(
		modify.UnregisterProvider(moduleImport, provider),
		modify.WhenFileExists(configPath, modify.File(configPath).Remove()),
	).Execute()
}
