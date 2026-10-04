package creatorflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type InfraiError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *InfraiError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewInfraiClient(apiKey, baseURL string) *InfraiClient {
	return &InfraiClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type AuthPhoneSendCodeRequest struct {
	Phone   string `json:"phone"`
	Purpose string `json:"purpose,omitempty"`
	Locale  string `json:"locale,omitempty"`
}

type AuthPhoneVerifyRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
	Login bool   `json:"login,omitempty"`
}

type AuthIdentityResolveRequest struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type AuthIdentityResolveResponse struct {
	UserID string `json:"user_id"`
}

type AuthSessionCreateRequest struct {
	UserID     string `json:"user_id"`
	Method     string `json:"method,omitempty"`
	MFAFactor  string `json:"mfa_factor,omitempty"`
	RequireMFA bool   `json:"require_mfa,omitempty"`
}

type AuthSessionCreateResponse struct {
	SessionID    string `json:"session_id"`
	RefreshToken string `json:"refresh_token"`
}

type SMSOTPRequest struct {
	To string `json:"to"`
}

func (c *InfraiClient) AuthPhoneSendCode(ctx context.Context, reqBody AuthPhoneSendCodeRequest) error {
	return c.post(ctx, "/v1/auth/phone/send_code", reqBody, nil)
}

func (c *InfraiClient) AuthPhoneVerify(ctx context.Context, reqBody AuthPhoneVerifyRequest) error {
	return c.post(ctx, "/v1/auth/phone/verify", reqBody, nil)
}

func (c *InfraiClient) AuthIdentityResolve(ctx context.Context, reqBody AuthIdentityResolveRequest) (AuthIdentityResolveResponse, error) {
	var out AuthIdentityResolveResponse
	err := c.post(ctx, "/v1/auth/identity/resolve", reqBody, &out)
	return out, err
}

func (c *InfraiClient) AuthSessionCreate(ctx context.Context, reqBody AuthSessionCreateRequest) (AuthSessionCreateResponse, error) {
	var out AuthSessionCreateResponse
	err := c.post(ctx, "/v1/auth/session/create", reqBody, &out)
	return out, err
}

func (c *InfraiClient) SMSOTP(ctx context.Context, reqBody SMSOTPRequest) error {
	return c.post(ctx, "/v1/sms/otp", reqBody, nil)
}

func (c *InfraiClient) post(ctx context.Context, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := c.baseURL + path
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		err = c.decodeEnvelope(resp, out)
		resp.Body.Close()
		if err == nil {
			return nil
		}

		var infraErr *InfraiError
		if errors.As(err, &infraErr) && infraErr.StatusCode == http.StatusTooManyRequests {
			wait := retryDelay(resp.Header.Get("Retry-After"), attempt)
			time.Sleep(wait)
			lastErr = err
			continue
		}
		return err
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("request failed")
}

func (c *InfraiClient) decodeEnvelope(resp *http.Response, out any) error {
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var env envelope
	if err := json.Unmarshal(payload, &env); err == nil && (env.OK || env.Error != nil || len(env.Data) > 0 || len(env.Metadata) > 0) {
		if !env.OK {
			msg := "request failed"
			code := "INFRAI_ERROR"
			if env.Error != nil {
				if env.Error.Message != "" {
					msg = env.Error.Message
				}
				if env.Error.Code != "" {
					code = env.Error.Code
				}
			}
			return &InfraiError{Code: code, Message: msg, StatusCode: resp.StatusCode}
		}
		if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return err
			}
		}
		return nil
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf("upstream server error: %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return &InfraiError{Code: "HTTP_ERROR", Message: string(payload), StatusCode: resp.StatusCode}
	}
	return nil
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return time.Duration(1<<attempt) * 200 * time.Millisecond
}
