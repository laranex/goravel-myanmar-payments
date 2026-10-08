package payments

import (
	"bytes"
	"errors"
	nethttp "net/http"
	"strings"

	"github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
)

// CallbackRequestFromContext turns the current Goravel request into the request the
// SDK verifies: the raw body, the headers and the query string.
//
// Goravel's gin driver parses urlencoded and multipart bodies before the handler
// runs, which consumes them. In that case the body is rebuilt from the parsed form
// fields (urlencoded, Content-Type set accordingly); the gateways that post forms
// (AYA Pay, CyberSource) verify field values, so the result verifies the same. JSON
// bodies (KBZ Pay, Wave Money, Yoma MMQR) are passed through byte for byte.
func CallbackRequestFromContext(ctx http.Context) (*myanmarpayments.CallbackRequest, error) {
	if ctx == nil || ctx.Request() == nil || ctx.Request().Origin() == nil {
		return nil, errors.New("goravel-myanmar-payments: the context has no request")
	}
	origin := ctx.Request().Origin()

	request, err := myanmarpayments.NewCallbackRequestFromHTTP(origin)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(request.Body)) == 0 && len(origin.PostForm) > 0 {
		request.Body = []byte(origin.PostForm.Encode())
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return request, nil
}

// Acknowledge returns the response the gateway expects after it delivers callback
// (status, body and headers of callback.Acknowledgement, e.g. KBZ Pay's plain
// "success"), so it stops retrying. A nil callback sends an empty 200.
func Acknowledge(ctx http.Context, callback *myanmarpayments.PaymentCallback) http.Response {
	acknowledgement := myanmarpayments.DefaultAcknowledgement()
	if callback != nil {
		acknowledgement = callback.Acknowledgement
	}

	contentType := "text/plain"
	for name, value := range acknowledgement.Headers {
		if strings.EqualFold(name, "Content-Type") {
			contentType = value

			continue
		}
		ctx.Response().Header(name, value)
	}

	status := acknowledgement.Status
	if status == 0 {
		status = nethttp.StatusOK
	}

	return ctx.Response().Data(status, contentType, []byte(acknowledgement.Body))
}
