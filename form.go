package payments

import (
	"encoding/json"
	"errors"
	nethttp "net/http"
	"net/url"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
)

// FormRouteName is the name of the auto-submit form route.
const FormRouteName = "myanmar-payments.form"

var (
	// ErrFormRouteDisabled is returned by AutoSubmitURL when
	// myanmar_payments.form_route.enabled is false.
	ErrFormRouteDisabled = errors.New("goravel-myanmar-payments: the form route is disabled (myanmar_payments.form_route.enabled)")
	// ErrCryptNotAvailable is returned by AutoSubmitURL when the crypt facade is not registered.
	ErrCryptNotAvailable = errors.New("goravel-myanmar-payments: the crypt facade is required to build form links; register &crypt.ServiceProvider{} and set APP_KEY")
	// ErrInvalidFormLink is returned by ResolveFormPayment for a tampered, undecryptable
	// or expired link.
	ErrInvalidFormLink = errors.New("goravel-myanmar-payments: this payment link is invalid or has expired")
)

// formLink is the encrypted payload of a form link.
type formLink struct {
	OrderID   string      `json:"orderId"`
	Action    string      `json:"action"`
	Fields    []formField `json:"fields"`
	Enctype   string      `json:"enctype"`
	ExpiresAt int64       `json:"expiresAt"`
}

type formField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AutoSubmitURL returns a link to the form route that renders form and posts it to the
// gateway from the customer's browser, so a handler can simply redirect to it. The
// form is encrypted with Goravel's crypt facade (APP_KEY) and the link expires after
// myanmar_payments.form_route.ttl_minutes, which is required: a missing or invalid
// value returns a *myanmarpayments.ConfigurationError.
//
// AYA Pay and CyberSource return a *myanmarpayments.FormPayment from Initiate; the Go
// SDK has no auto-submit URL of its own, so this package adds it.
func (m *Manager) AutoSubmitURL(form *myanmarpayments.FormPayment) (string, error) {
	if form == nil {
		return "", errors.New("goravel-myanmar-payments: the form payment is nil")
	}
	if !m.settings.FormRouteEnabled() {
		return "", ErrFormRouteDisabled
	}
	if m.crypt == nil {
		return "", ErrCryptNotAvailable
	}

	ttl, err := m.settings.FormRouteTTL()
	if err != nil {
		return "", err
	}

	link := formLink{
		OrderID:   form.OrderID,
		Action:    form.Action,
		Fields:    make([]formField, len(form.Fields)),
		Enctype:   form.Enctype,
		ExpiresAt: m.now().Add(ttl).Unix(),
	}
	for i, field := range form.Fields {
		link.Fields[i] = formField{Name: field.Name, Value: field.Value}
	}

	encoded, err := json.Marshal(link)
	if err != nil {
		return "", err
	}
	payload, err := m.crypt.EncryptString(string(encoded))
	if err != nil {
		return "", err
	}

	return m.settings.FormRouteBaseURL() + m.settings.FormRoutePath() + "?" + url.Values{"payload": {payload}}.Encode(), nil
}

// ResolveFormPayment decrypts the payload of a form link. It returns ErrInvalidFormLink
// when the payload was tampered with, was encrypted with another key or has expired.
func (m *Manager) ResolveFormPayment(payload string) (*myanmarpayments.FormPayment, error) {
	if payload == "" || m.crypt == nil {
		return nil, ErrInvalidFormLink
	}
	decrypted, err := m.crypt.DecryptString(payload)
	if err != nil {
		return nil, ErrInvalidFormLink
	}

	var link formLink
	if err := json.Unmarshal([]byte(decrypted), &link); err != nil || link.Action == "" || link.ExpiresAt < m.now().Unix() {
		return nil, ErrInvalidFormLink
	}

	form := &myanmarpayments.FormPayment{
		OrderID: link.OrderID,
		Action:  link.Action,
		Fields:  make([]myanmarpayments.FormField, len(link.Fields)),
		Enctype: link.Enctype,
	}
	for i, field := range link.Fields {
		form.Fields[i] = myanmarpayments.FormField{Name: field.Name, Value: field.Value}
	}

	return form, nil
}

// ServeForm is the handler of the form route: it renders the form behind the payload
// query parameter as a page that submits itself (FormPayment.HTML), or answers 410
// Gone for an invalid or expired link.
func (m *Manager) ServeForm(ctx http.Context) http.Response {
	form, err := m.ResolveFormPayment(ctx.Request().Query("payload"))
	if err != nil {
		return ctx.Response().String(nethttp.StatusGone, "This payment link is invalid or has expired.")
	}

	ctx.Response().Header("Cache-Control", "no-store")
	ctx.Response().Header("Referrer-Policy", "no-referrer-when-downgrade")

	return ctx.Response().Data(nethttp.StatusOK, "text/html; charset=utf-8", []byte(form.HTML()))
}

// registerFormRoute registers GET myanmar_payments.form_route.path on router, with the
// configured middleware, answering with handler.
func registerFormRoute(router route.Router, s settings, handler http.HandlerFunc) route.Action {
	if middleware := s.FormRouteMiddleware(); len(middleware) > 0 {
		router = router.Middleware(middleware...)
	}

	return router.Get(s.FormRoutePath(), handler).Name(FormRouteName)
}
