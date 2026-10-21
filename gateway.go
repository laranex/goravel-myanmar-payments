package payments

import (
	"errors"
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/wavepayminiapp"
)

// The gateway names Manager.Gateway and Manager.HandleCallback accept, e.g. as
// the {gateway} segment of one callback route that serves every gateway.
const (
	GatewayKbzPay         = "kbz-pay"
	GatewayWavePay        = "wave-pay"
	GatewayWavePayMiniApp = "wave-pay-mini-app"
	GatewayAyaPay         = "aya-pay"
	GatewayYomaMmqr       = "yoma-mmqr"
	GatewayCyberSource    = "cyber-source"
)

// ErrUnknownGateway is returned by Manager.Gateway and Manager.HandleCallback for
// a name that is not one of GatewayNames.
var ErrUnknownGateway = errors.New("goravel-myanmar-payments: unknown payment gateway")

// ErrCallbackURLRequired is returned when a Wave Pay Mini App callback is verified
// without the callback URL its payment was signed with: by Manager.HandleCallback
// and by the handler Manager.Gateway returns while its CallbackURL is empty. Use
// Manager.HandleCallbackFromContext, set WavePayMiniAppCallbackHandler.CallbackURL,
// or call WavePayMiniApp().HandleCallback(request, callbackURL).
var ErrCallbackURLRequired = errors.New("goravel-myanmar-payments: the wave-pay-mini-app callback needs the callback URL the payment was signed with; use HandleCallbackFromContext or WavePayMiniApp().HandleCallback(request, callbackURL)")

// GatewayNames returns every gateway name, in the order of the manager's accessors.
func GatewayNames() []string {
	return []string{GatewayKbzPay, GatewayWavePay, GatewayWavePayMiniApp, GatewayAyaPay, GatewayYomaMmqr, GatewayCyberSource}
}

// CallbackHandler is what every SDK gateway has in common: it verifies a callback
// and returns the typed result. Manager.Gateway returns the SDK gateway itself
// (e.g. *kbzpay.Gateway) behind this interface, except for Wave Pay Mini App,
// whose HandleCallback also takes a callback URL: it returns a
// WavePayMiniAppCallbackHandler.
type CallbackHandler interface {
	HandleCallback(request *myanmarpayments.CallbackRequest) (*myanmarpayments.PaymentCallback, error)
}

// Gateway returns the gateway for name (GatewayKbzPay, "wave-pay", ...). An
// unknown name returns an error wrapping ErrUnknownGateway; a missing credential
// returns a *myanmarpayments.ConfigurationError, as the accessors do. For
// "wave-pay-mini-app" it returns a WavePayMiniAppCallbackHandler with no
// CallbackURL.
func (m *Manager) Gateway(name string) (CallbackHandler, error) {
	switch name {
	case GatewayKbzPay:
		return handler(m.KbzPay())
	case GatewayWavePay:
		return handler(m.WavePay())
	case GatewayWavePayMiniApp:
		gateway, err := m.WavePayMiniApp()
		if err != nil {
			return nil, err
		}

		return &WavePayMiniAppCallbackHandler{Gateway: gateway}, nil
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
//
// A CallbackRequest does not carry the URL it was posted to, so "wave-pay-mini-app"
// returns ErrCallbackURLRequired (after resolving the gateway): verify those with
// HandleCallbackFromContext, or with WavePayMiniApp().HandleCallback(request,
// callbackURL), e.g. a stored request with the URL stored beside it.
func (m *Manager) HandleCallback(name string, request *myanmarpayments.CallbackRequest) (*myanmarpayments.PaymentCallback, error) {
	gateway, err := m.Gateway(name)
	if err != nil {
		return nil, err
	}

	return gateway.HandleCallback(request)
}

// HandleCallbackFromContext verifies the callback in ctx with the named gateway, e.g.
// in one route that serves every gateway. It resolves the gateway first (an error
// wrapping ErrUnknownGateway or a *myanmarpayments.ConfigurationError), then reads
// the request with CallbackRequestFromContext (ErrCallbackBodyTooLarge), then
// verifies it (*myanmarpayments.SignatureVerificationError). Wave Pay Mini App
// callbacks are verified against CallbackURLFromContext(ctx), the URL they were
// posted to.
func (m *Manager) HandleCallbackFromContext(name string, ctx http.Context) (*myanmarpayments.PaymentCallback, error) {
	gateway, err := m.Gateway(name)
	if err != nil {
		return nil, err
	}
	request, err := CallbackRequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if miniApp, ok := gateway.(*WavePayMiniAppCallbackHandler); ok {
		return miniApp.Gateway.HandleCallback(request, CallbackURLFromContext(ctx))
	}

	return gateway.HandleCallback(request)
}

// WavePayMiniAppCallbackHandler is the CallbackHandler Manager.Gateway returns for
// "wave-pay-mini-app". Wave Pay Mini App callbacks are signed with the callback URL
// of the payment, which the callback itself does not carry.
type WavePayMiniAppCallbackHandler struct {
	// Gateway is the SDK gateway (Manager.WavePayMiniApp()).
	Gateway *wavepayminiapp.Gateway
	// CallbackURL is the callback URL the payment was signed with. Empty returns
	// ErrCallbackURLRequired.
	CallbackURL string
}

// HandleCallback verifies request against CallbackURL.
func (h *WavePayMiniAppCallbackHandler) HandleCallback(request *myanmarpayments.CallbackRequest) (*myanmarpayments.PaymentCallback, error) {
	if h.CallbackURL == "" {
		return nil, ErrCallbackURLRequired
	}

	return h.Gateway.HandleCallback(request, h.CallbackURL)
}

// handler returns gateway as a CallbackHandler, keeping a nil interface on error.
func handler[T CallbackHandler](gateway T, err error) (CallbackHandler, error) {
	if err != nil {
		return nil, err
	}

	return gateway, nil
}
