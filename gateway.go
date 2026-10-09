package payments

import (
	"errors"
	"fmt"
	"strings"

	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
)

// The gateway names Manager.Gateway and Manager.HandleCallback accept, e.g. as
// the {gateway} segment of one callback route that serves every gateway.
const (
	GatewayKbzPay      = "kbz-pay"
	GatewayWaveMoney   = "wave-money"
	GatewayAyaPay      = "aya-pay"
	GatewayYomaMmqr    = "yoma-mmqr"
	GatewayCyberSource = "cyber-source"
)

// ErrUnknownGateway is returned by Manager.Gateway and Manager.HandleCallback for
// a name that is not one of GatewayNames.
var ErrUnknownGateway = errors.New("goravel-myanmar-payments: unknown payment gateway")

// GatewayNames returns every gateway name, in the order of the manager's accessors.
func GatewayNames() []string {
	return []string{GatewayKbzPay, GatewayWaveMoney, GatewayAyaPay, GatewayYomaMmqr, GatewayCyberSource}
}

// CallbackHandler is what every SDK gateway has in common: it verifies a callback
// and returns the typed result. Manager.Gateway returns the SDK gateway itself
// (e.g. *kbzpay.Gateway) behind this interface.
type CallbackHandler interface {
	HandleCallback(request *myanmarpayments.CallbackRequest) (*myanmarpayments.PaymentCallback, error)
}

// Gateway returns the gateway for name (GatewayKbzPay, "wave-money", ...). An
// unknown name returns an error wrapping ErrUnknownGateway; a missing credential
// returns a *myanmarpayments.ConfigurationError, as the accessors do.
func (m *Manager) Gateway(name string) (CallbackHandler, error) {
	switch name {
	case GatewayKbzPay:
		return handler(m.KbzPay())
	case GatewayWaveMoney:
		return handler(m.WaveMoney())
	case GatewayAyaPay:
		return handler(m.AyaPay())
	case GatewayYomaMmqr:
		return handler(m.YomaMmqr())
	case GatewayCyberSource:
		return handler(m.CyberSource())
	default:
		return nil, fmt.Errorf("%w [%s]; use one of %s", ErrUnknownGateway, name, strings.Join(GatewayNames(), ", "))
	}
}

// HandleCallback verifies request with the named gateway, e.g. in one route that
// serves every gateway. A bad signature returns a
// *myanmarpayments.SignatureVerificationError.
func (m *Manager) HandleCallback(name string, request *myanmarpayments.CallbackRequest) (*myanmarpayments.PaymentCallback, error) {
	gateway, err := m.Gateway(name)
	if err != nil {
		return nil, err
	}

	return gateway.HandleCallback(request)
}

// handler returns gateway as a CallbackHandler, keeping a nil interface on error.
func handler[T CallbackHandler](gateway T, err error) (CallbackHandler, error) {
	if err != nil {
		return nil, err
	}

	return gateway, nil
}
