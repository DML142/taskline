package project

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathUUIDRejectsInvalidValueOnce(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/invalid", nil)
	request.SetPathValue("workspaceID", "invalid")

	_, ok := pathUUID(response, request, "workspaceID")

	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.JSONEq(t, `{"error":{"code":"invalid_request","message":"Invalid request"}}`, response.Body.String())
}

func TestDecodeJSONRejectsTrailingContent(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Website"} trailing`))
	var target projectRequest

	ok := decodeJSON(response, request, &target)

	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.JSONEq(t, `{"error":{"code":"invalid_request","message":"Invalid request body"}}`, response.Body.String())
}
