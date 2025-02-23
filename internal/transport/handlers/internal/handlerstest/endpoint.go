package handlerstest

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Want struct {
	Status      int
	Body        string
	ContentType string
}

type TCase struct {
	Name        string
	Token       string
	ContentType string
	Body        any
	Want
}

func TestEndpoint(t *testing.T, server *httptest.Server, cases []TCase, method string, url string) {
	client := resty.New().SetBaseURL(server.URL)

	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			response, err := client.R().
				SetBody(tt.Body).
				SetHeader("Content-Type", tt.ContentType).
				SetAuthToken(tt.Token).
				Execute(method, url)

			require.NoError(t, err)

			assert.Equal(t, tt.Want.Status, response.StatusCode())

			if tt.Want.ContentType != "" {
				assert.Equal(t, tt.Want.ContentType, response.Header().Get("Content-Type"))
			}

			if tt.Want.Body != "" {
				if strings.Contains(tt.Want.ContentType, "json") {
					assert.JSONEq(t, tt.Want.Body, string(response.Body()))
				} else {
					assert.Contains(t, string(response.Body()), tt.Want.Body)
				}
			}
		})
	}
}
