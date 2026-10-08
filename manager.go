package payments

import (
	"errors"
	"sync"
	"time"

	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/contracts/crypt"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/ayapay"
	"github.com/laranex/go-myanmar-payments/v4/cybersource"
	"github.com/laranex/go-myanmar-payments/v4/kbzpay"
	"github.com/laranex/go-myanmar-payments/v4/wavemoney"
	"github.com/laranex/go-myanmar-payments/v4/yomammqr"
)

// Options are the dependencies of a Manager. The ServiceProvider fills them from the
// Goravel container.
type Options struct {
	// Config holds myanmar_payments.* and the environment. Required.
	Config config.Config
	// HTTPClient sends gateway requests; nil uses the SDK's default client.
	HTTPClient myanmarpayments.HTTPDoer
	// TokenCache keeps Yoma MMQR access tokens; nil uses the SDK's in-memory cache.
	TokenCache myanmarpayments.TokenCache
	// Crypt encrypts the auto-submit form links; nil disables AutoSubmitURL.
	Crypt crypt.Crypt
	// Now returns the current time; nil uses time.Now.
	Now func() time.Time
}

// Manager is the entry point to every gateway. Each gateway is built from the
// configuration the first time it is requested and then reused; it is safe for
// concurrent use.
type Manager struct {
	settings   settings
	httpClient myanmarpayments.HTTPDoer
	tokenCache myanmarpayments.TokenCache
	crypt      crypt.Crypt
	now        func() time.Time

	mu          sync.Mutex
	kbzPay      *kbzpay.Gateway
	waveMoney   *wavemoney.Gateway
	ayaPay      *ayapay.Gateway
	yomaMmqr    *yomammqr.Gateway
	cyberSource *cybersource.Gateway
}

// NewManager returns a Manager. Most applications use the one bound by the
// ServiceProvider (facades.MyanmarPayments()) instead.
func NewManager(options Options) (*Manager, error) {
	if options.Config == nil {
		return nil, errors.New("goravel-myanmar-payments: a config.Config is required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}

	return &Manager{
		settings:   settings{config: options.Config},
		httpClient: options.HTTPClient,
		tokenCache: options.TokenCache,
		crypt:      options.Crypt,
		now:        options.Now,
	}, nil
}

// KbzPay returns the KBZ Pay gateway: PWA, QR, App, Status and HandleCallback.
// A missing credential returns a *myanmarpayments.ConfigurationError.
func (m *Manager) KbzPay() (*kbzpay.Gateway, error) {
	return lazy(&m.mu, &m.kbzPay, func() (*kbzpay.Gateway, error) {
		return kbzpay.New(kbzpay.ConfigFromEnv(m.settings.Getenv), m.httpClient)
	})
}

// WaveMoney returns the Wave Money gateway: Initiate and HandleCallback.
func (m *Manager) WaveMoney() (*wavemoney.Gateway, error) {
	return lazy(&m.mu, &m.waveMoney, func() (*wavemoney.Gateway, error) {
		return wavemoney.New(wavemoney.ConfigFromEnv(m.settings.Getenv), m.httpClient)
	})
}

// AyaPay returns the AYA Payment Gateway: Services, Initiate, Status, HandleCallback
// and VerifyRedirect. Pass the form Initiate returns to AutoSubmitURL.
func (m *Manager) AyaPay() (*ayapay.Gateway, error) {
	return lazy(&m.mu, &m.ayaPay, func() (*ayapay.Gateway, error) {
		return ayapay.New(ayapay.ConfigFromEnv(m.settings.Getenv), m.httpClient)
	})
}

// YomaMmqr returns the Yoma MMQR gateway: Initiate, RenewQR, Status, HandleCallback
// and ForgetToken. Its access tokens live in the configured cache store.
func (m *Manager) YomaMmqr() (*yomammqr.Gateway, error) {
	return lazy(&m.mu, &m.yomaMmqr, func() (*yomammqr.Gateway, error) {
		return yomammqr.New(yomammqr.ConfigFromEnv(m.settings.Getenv), m.httpClient, m.tokenCache)
	})
}

// CyberSource returns the CyberSource Secure Acceptance gateway: Initiate and
// HandleCallback. Pass the form Initiate returns to AutoSubmitURL.
func (m *Manager) CyberSource() (*cybersource.Gateway, error) {
	return lazy(&m.mu, &m.cyberSource, func() (*cybersource.Gateway, error) {
		return cybersource.New(cybersource.ConfigFromEnv(m.settings.Getenv))
	})
}

// lazy builds *slot once; a failed build is not cached, so fixing the configuration
// takes effect on the next call.
func lazy[T any](mu *sync.Mutex, slot **T, build func() (*T, error)) (*T, error) {
	mu.Lock()
	defer mu.Unlock()

	if *slot == nil {
		gateway, err := build()
		if err != nil {
			return nil, err
		}
		*slot = gateway
	}

	return *slot, nil
}
