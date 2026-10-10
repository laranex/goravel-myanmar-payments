package payments

import (
	nethttp "net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/ayapay"
	"github.com/laranex/go-myanmar-payments/v4/cybersource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

type formServer struct {
	*testServer
	manager *Manager
	clock   *clock
}

func newFormServer(t *testing.T, formRoute map[string]any) *formServer {
	t.Helper()
	section := credentials("https://gateway.test")
	for key, value := range formRoute {
		section["form_route"].(map[string]any)[key] = value
	}
	config := newConfig(t, map[string]any{ConfigKey: section})
	c := &clock{now: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)}
	manager := newManager(t, config, nil, c.Now)
	server := &formServer{testServer: newTestServer(config), manager: manager, clock: c}
	registerFormRoute(server.router, manager.settings, manager.ServeForm)

	return server
}

// open follows an auto-submit URL on the test server.
func (s *formServer) open(t *testing.T, link string) *nethttp.Response {
	t.Helper()
	parsed, err := url.Parse(link)
	require.NoError(t, err)

	return s.do(t, nethttp.MethodGet, parsed.RequestURI(), "", "")
}

func ayaForm(t *testing.T, manager *Manager) *myanmarpayments.FormPayment {
	t.Helper()
	aya, err := manager.AyaPay()
	require.NoError(t, err)
	form, err := aya.Initiate(ayapay.PaymentData{
		OrderID: "ORD123456", Amount: myanmarpayments.Kyat(1000), Channel: "aya_pay", Method: ayapay.MethodQR,
		ReturnURL: "https://shop.test/aya/done", Description: `Tea & "cake"`,
	})
	require.NoError(t, err)

	return form
}

func TestAutoSubmitURLRendersTheSignedForm(t *testing.T) {
	server := newFormServer(t, nil)
	form := ayaForm(t, server.manager)

	link, err := server.manager.AutoSubmitURL(form)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(link, "https://shop.test/myanmar-payments/form?payload="))

	response := server.open(t, link)
	body := readBody(t, response)
	assert.Equal(t, nethttp.StatusOK, response.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", response.Header.Get("Content-Type"))
	assert.Equal(t, "no-store", response.Header.Get("Cache-Control"))
	assert.Equal(t, "no-referrer-when-downgrade", response.Header.Get("Referrer-Policy"))
	assert.Equal(t, form.HTML(), body, "the page is the SDK's own form HTML")
	assert.Contains(t, body, `action="https://gateway.test/aya/v1/payment/request"`)
	assert.Contains(t, body, `enctype="multipart/form-data"`)
	checkSum, _ := form.Field("checkSum")
	assert.Contains(t, body, `name="checkSum" value="`+checkSum+`"`)
}

func TestResolveFormPaymentKeepsEveryField(t *testing.T) {
	server := newFormServer(t, nil)
	cyber, err := server.manager.CyberSource()
	require.NoError(t, err)
	form, err := cyber.Initiate(cybersource.PaymentData{
		OrderID: "ORD-1", Amount: myanmarpayments.Kyat(1000), CallbackURL: "https://shop.test/callback",
		Currency: "MMK", TransactionType: cybersource.Sale, Locale: "en-us",
		ReturnURL: "https://shop.test/done", CancelURL: "https://shop.test/cancel",
	})
	require.NoError(t, err)
	assert.Equal(t, "application/x-www-form-urlencoded", form.Enctype)

	link, err := server.manager.AutoSubmitURL(form)
	require.NoError(t, err)
	parsed, err := url.Parse(link)
	require.NoError(t, err)

	resolved, err := server.manager.ResolveFormPayment(parsed.Query().Get("payload"))
	require.NoError(t, err)
	assert.Equal(t, form, resolved)
	assert.Contains(t, readBody(t, server.open(t, link)), `enctype="application/x-www-form-urlencoded"`)
}

func TestExpiredFormLinksAreGone(t *testing.T) {
	server := newFormServer(t, map[string]any{"ttl_minutes": 5})
	link, err := server.manager.AutoSubmitURL(ayaForm(t, server.manager))
	require.NoError(t, err)

	server.clock.now = server.clock.now.Add(5 * time.Minute)
	assert.Equal(t, nethttp.StatusOK, server.open(t, link).StatusCode, "valid until the last second")

	server.clock.now = server.clock.now.Add(time.Second)
	response := server.open(t, link)
	assert.Equal(t, nethttp.StatusGone, response.StatusCode)
	assert.Equal(t, "This payment link is invalid or has expired.", readBody(t, response))
}

func TestTamperedFormLinksAreGone(t *testing.T) {
	server := newFormServer(t, nil)
	link, err := server.manager.AutoSubmitURL(ayaForm(t, server.manager))
	require.NoError(t, err)
	parsed, err := url.Parse(link)
	require.NoError(t, err)
	payload := parsed.Query().Get("payload")

	other := newConfig(t, nil)
	other.Add("app", map[string]any{"key": "zyxwvutsrqponmlkjihgfedcba654321"})
	foreign, err := newCrypt(t, other).EncryptString(`{"action":"https://evil.test","fields":[],"expiresAt":9999999999}`)
	require.NoError(t, err)

	for name, value := range map[string]string{
		"flipped character": payload[:len(payload)-3] + "AAA",
		"not base64":        "not-a-payload",
		"empty":             "",
		"other app key":     foreign,
		"truncated":         payload[:len(payload)/2],
	} {
		response := server.do(t, nethttp.MethodGet, "/myanmar-payments/form?"+url.Values{"payload": {value}}.Encode(), "", "")
		assert.Equal(t, nethttp.StatusGone, response.StatusCode, name)
		_, err := server.manager.ResolveFormPayment(value)
		assert.ErrorIs(t, err, ErrInvalidFormLink, name)
	}
	assert.Equal(t, nethttp.StatusGone, server.do(t, nethttp.MethodGet, "/myanmar-payments/form", "", "").StatusCode)
}

func TestFormRouteOptions(t *testing.T) {
	var hits int
	middleware := countingMiddleware{hits: &hits}
	server := newFormServer(t, map[string]any{"path": "/pay/form/", "base_url": "https://pay.test/", "middleware": []contractshttp.Middleware{middleware}})

	link, err := server.manager.AutoSubmitURL(ayaForm(t, server.manager))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(link, "https://pay.test/pay/form?payload="))
	assert.Equal(t, nethttp.StatusOK, server.open(t, link).StatusCode)
	assert.Equal(t, 1, hits)
}

func TestAutoSubmitURLErrors(t *testing.T) {
	t.Run("disabled route", func(t *testing.T) {
		server := newFormServer(t, map[string]any{"enabled": false})
		_, err := server.manager.AutoSubmitURL(ayaForm(t, server.manager))
		assert.ErrorIs(t, err, ErrFormRouteDisabled)
	})
	t.Run("no crypt", func(t *testing.T) {
		manager, err := NewManager(Options{Config: newConfig(t, nil)})
		require.NoError(t, err)
		_, err = manager.AutoSubmitURL(&myanmarpayments.FormPayment{Action: "https://gateway.test"})
		assert.ErrorIs(t, err, ErrCryptNotAvailable)
		_, err = manager.ResolveFormPayment("payload")
		assert.ErrorIs(t, err, ErrInvalidFormLink)
	})
	t.Run("missing ttl", func(t *testing.T) {
		server := newFormServer(t, map[string]any{"ttl_minutes": ""})
		_, err := server.manager.AutoSubmitURL(ayaForm(t, server.manager))
		assert.EqualError(t, err, "myanmarpayments: The form_route configuration is missing [ttl_minutes].")
	})
	t.Run("invalid ttl", func(t *testing.T) {
		server := newFormServer(t, map[string]any{"ttl_minutes": "0"})
		_, err := server.manager.AutoSubmitURL(ayaForm(t, server.manager))
		assert.EqualError(t, err, "myanmarpayments: The form_route configuration [ttl_minutes] must be a whole number greater than 0.")
	})
	t.Run("nil form", func(t *testing.T) {
		server := newFormServer(t, nil)
		_, err := server.manager.AutoSubmitURL(nil)
		assert.ErrorContains(t, err, "form payment is nil")
	})
}

type countingMiddleware struct{ hits *int }

func (m countingMiddleware) Signature() string { return "counting" }

func (m countingMiddleware) Handle(ctx contractshttp.Context) {
	*m.hits++
	ctx.Request().Next()
}
