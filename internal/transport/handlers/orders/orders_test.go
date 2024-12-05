package orders

import (
	"net/http"
	"testing"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/internal/test"
)

const orders = "/api/user/orders"

func TestAuth(t *testing.T) {
	log.InitTestLogger(t)

	t.Run("upload", testUpload)
}

func testUpload(t *testing.T) {
	server, token := test.NewTestServerWithLoggedInUser(t)
	defer server.Close()

	tests := []test.TCase{
		{
			Name:        "success",
			Token:       token,
			ContentType: "text/plain",
			Body:        "number",
			Want: test.Want{
				Status: http.StatusAccepted,
			},
		},
	}

	test.TestEndpoint(t, server, tests, http.MethodPost, orders)
}
