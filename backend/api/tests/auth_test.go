package tests

import (
	"Server/models"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeJSONRequest(t *testing.T, method, path string, payload interface{}) (*http.Response, interface{}) {
	t.Helper()

	var requestBody *bytes.Buffer
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)
		requestBody = bytes.NewBuffer(payloadBytes)
	}

	req, err := http.NewRequest(method, path, requestBody)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	var responseBody interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	require.NoError(t, err)

	return resp, responseBody
}

func TestUserRegistration(t *testing.T) {
	// Clear the collection before the test.
	cleanupCollections()

	tests := []struct {
		name           string
		payload        models.CreateUser
		expectedStatus int
		shouldContain  []string
	}{
		{
			name: "Valid Registration",
			payload: models.CreateUser{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			expectedStatus: http.StatusCreated,
			shouldContain:  []string{"result", "token"},
		},
		{
			name: "Missing Required Fields",
			payload: models.CreateUser{
				Email: "missing@example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate Email Registration",
			payload: models.CreateUser{
				Email:     "duplicate@example.com",
				Password:  "password456",
				FirstName: "jane",
				LastName:  "smith",
			},
			expectedStatus: http.StatusConflict,
			shouldContain:  []string{"already exists"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the duplicate-email test, register the original user first.
			if tt.name == "Duplicate Email Registration" {
				firstPayload := models.CreateUser{
					Email:     "duplicate@example.com",
					Password:  "passwo2226",
					FirstName: "ahmed",
					LastName:  "khalaf",
				}
				registerUser(t, firstPayload, http.StatusCreated)
			}

			resp, responseBody := makeJSONRequest(t, http.MethodPost, "/user/signup", tt.payload)

			// Check the status code.
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Convert to string for contains checks.
			responseStr, _ := json.Marshal(responseBody)
			// Check expected content.
			for _, contain := range tt.shouldContain {
				assert.Contains(t, string(responseStr), contain)
			}

			// For successful registration, verify the basic structure.
			if tt.expectedStatus == http.StatusCreated {
				if respMap, ok := responseBody.(map[string]interface{}); ok {
					assert.Contains(t, respMap, "token")
					assert.Contains(t, respMap, "result")

					if result, ok := respMap["result"].(map[string]interface{}); ok {
						assert.Equal(t, tt.payload.Email, result["email"])
						expectedName := tt.payload.FirstName + " " + tt.payload.LastName
						assert.Equal(t, expectedName, result["name"])
						assert.NotContains(t, result, "password", "password hash must never be present in an API response")
					}
				}
			}
		})
	}
}

func TestUserLogin(t *testing.T) {
	cleanupCollections()

	// Register a user first.
	registerPayload := models.CreateUser{
		Email:     "login@example.com",
		Password:  "password123",
		FirstName: "Login",
		LastName:  "Test",
	}

	registerUser(t, registerPayload, http.StatusCreated)

	tests := []struct {
		name           string
		payload        models.LoginUser
		expectedStatus int
		shouldContain  []string
	}{
		{
			name: "Valid Login",
			payload: models.LoginUser{
				Email:    "login@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusCreated,
			shouldContain:  []string{"result", "token"},
		},
		{
			name: "Invalid Email",
			payload: models.LoginUser{
				Email:    "noneexistent@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Invalid Password",
			payload: models.LoginUser{
				Email:    "login@example.com",
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Empty",
			payload: models.LoginUser{
				Email:    "",
				Password: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}
	// Loop through the test cases.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, responseBody := makeJSONRequest(t, http.MethodPost, "/user/signin", tt.payload)
			// Check the status code.
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Convert to string for contains checks.
			responseStr, _ := json.Marshal(responseBody)
			// Check expected content.
			for _, contain := range tt.shouldContain {
				assert.Contains(t, string(responseStr), contain)
			}

			// For successful login, verify the basic structure.
			if tt.expectedStatus == http.StatusCreated {
				if respMap, ok := responseBody.(map[string]interface{}); ok {
					assert.Contains(t, respMap, "token")
					assert.Contains(t, respMap, "result")

					if result, ok := respMap["result"].(map[string]interface{}); ok {
						assert.Equal(t, tt.payload.Email, result["email"])
						assert.NotContains(t, result, "password", "password hash must never be present in an API response")
					}
				}
			}

		})
	}
}

func TestRefreshUser(t *testing.T) {
	cleanupCollections()

	registerPayload := models.CreateUser{
		Email:     "refresh@example.com",
		Password:  "password123",
		FirstName: "Refresh",
		LastName:  "Test",
	}
	_, registerBody := makeJSONRequest(t, http.MethodPost, "/user/signup", registerPayload)
	registerMap, ok := registerBody.(map[string]interface{})
	require.True(t, ok)
	token, ok := registerMap["token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, token)

	t.Run("Missing Authorization header", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/user/refresh", nil)
		require.NoError(t, err)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Valid token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/user/refresh", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		var responseBody map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&responseBody))

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, responseBody, "token")
		assert.Contains(t, responseBody, "result")

		if result, ok := responseBody["result"].(map[string]interface{}); ok {
			assert.Equal(t, registerPayload.Email, result["email"])
			assert.NotContains(t, result, "password", "password hash must never be present in an API response")
		}
	})
}

// Helper function to create a user for testing.
func registerUser(t *testing.T, payload models.CreateUser, expectedStatus int) {
	t.Helper()
	resp, _ := makeJSONRequest(t, http.MethodPost, "/user/signup", payload)
	assert.Equal(t, expectedStatus, resp.StatusCode)
}
