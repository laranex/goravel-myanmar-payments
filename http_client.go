package payments

import (
	"fmt"
	"net/http"
	"time"

	"github.com/goravel/framework/contracts/http/client"
)

// HTTPClient is the myanmarpayments.HTTPDoer the gateways use inside a Goravel
// application. Every request goes through the *http.Client of a Goravel HTTP client
// (facades.Http()), resolved per request, so facades.Http().Fake(...) and
// AssertSent see gateway calls, and the client's transport settings apply.
type HTTPClient struct {
	factory  client.Factory
	name     string
	timeout  time.Duration
	fallback *http.Client
}

// NewHTTPClient returns an HTTPClient that uses factory's client named name (empty
// means the default client) with timeout; a zero timeout keeps the Goravel client's
// own. A nil factory sends requests with a plain *http.Client.
func NewHTTPClient(factory client.Factory, name string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{factory: factory, name: name, timeout: timeout, fallback: &http.Client{Timeout: timeout}}
}

// Do sends req.
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c.factory == nil {
		return c.fallback.Do(req)
	}

	base := c.factory.Client(c.name).HttpClient()
	if base == nil {
		return nil, fmt.Errorf("goravel-myanmar-payments: the Goravel HTTP client %q is not configured in http.clients", c.name)
	}

	// Copy the client so the timeout applies to gateway calls only. The transport,
	// which carries Goravel's fakes, is shared.
	httpClient := *base
	if c.timeout > 0 {
		httpClient.Timeout = c.timeout
	}

	return httpClient.Do(req)
}
