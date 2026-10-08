package facades

import (
	"testing"

	frameworkconfig "github.com/goravel/framework/config"
	mocksfoundation "github.com/goravel/framework/mocks/foundation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	payments "github.com/laranex/goravel-myanmar-payments/v4"
)

func TestMyanmarPaymentsPanicsWithoutTheServiceProvider(t *testing.T) {
	assert.PanicsWithError(t, "goravel-myanmar-payments: register &payments.ServiceProvider{} in bootstrap/providers.go", func() { MyanmarPayments() })
}

func TestMyanmarPaymentsResolvesTheBoundManager(t *testing.T) {
	t.Setenv("APP_KEY", "abcdefghijklmnopqrstuvwxyz123456")
	config := frameworkconfig.NewApplication("")
	config.Add("myanmar_payments", map[string]any{"cyber_source": map[string]any{"profile_id": "p", "access_key": "a", "secret_key": "s"}})
	manager, err := payments.NewManager(payments.Options{Config: config})
	require.NoError(t, err)

	app := mocksfoundation.NewApplication(t)
	app.EXPECT().Singleton(payments.Binding, mock.Anything).Once()
	app.EXPECT().Make(payments.Binding).Return(manager, nil).Once()
	(&payments.ServiceProvider{}).Register(app)

	assert.Same(t, manager, MyanmarPayments())

	cyber, err := manager.CyberSource()
	require.NoError(t, err)
	assert.Equal(t, "p", cyber.Config().ProfileID)
}
