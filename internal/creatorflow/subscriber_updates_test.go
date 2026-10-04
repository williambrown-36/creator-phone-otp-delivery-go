package creatorflow

import (
	"context"
	"testing"
)

type fakeInfrai struct {
	sendCodeCalls int
	verifyCalls   int
	smsCalls      int
	lastSMS       SMSOTPRequest
	identity      AuthIdentityResolveResponse
	session       AuthSessionCreateResponse
}

func (f *fakeInfrai) AuthPhoneSendCode(ctx context.Context, reqBody AuthPhoneSendCodeRequest) error {
	f.sendCodeCalls++
	return nil
}

func (f *fakeInfrai) AuthPhoneVerify(ctx context.Context, reqBody AuthPhoneVerifyRequest) error {
	f.verifyCalls++
	return nil
}

func (f *fakeInfrai) AuthIdentityResolve(ctx context.Context, reqBody AuthIdentityResolveRequest) (AuthIdentityResolveResponse, error) {
	return f.identity, nil
}

func (f *fakeInfrai) AuthSessionCreate(ctx context.Context, reqBody AuthSessionCreateRequest) (AuthSessionCreateResponse, error) {
	return f.session, nil
}

func (f *fakeInfrai) SMSOTP(ctx context.Context, reqBody SMSOTPRequest) error {
	f.smsCalls++
	f.lastSMS = reqBody
	return nil
}

func TestActivatePurchase(t *testing.T) {
	tests := []struct {
		name         string
		input        VerifyPurchaseInput
		expectStatus string
		expectActive bool
		expectSMSTo  string
	}{
		{
			name: "verified phone purchase activates subscriber and releases asset",
			input: VerifyPurchaseInput{
				Phone:     "+14155550123",
				Code:      "123456",
				Campaign:  "spring-drop",
				AssetID:   "beat-pack-01",
				AssetURL:  "https://cdn.example.com/downloads/beat-pack-01.zip",
				BuyerName: "Mina",
				Login:     true,
			},
			expectStatus: "delivered",
			expectActive: true,
			expectSMSTo:  "+14155550123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeInfrai{
				identity: AuthIdentityResolveResponse{UserID: "user_123"},
				session:  AuthSessionCreateResponse{SessionID: "sess_123"},
			}
			service := NewService(fake)

			result, err := service.ActivatePurchase(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("ActivatePurchase() error = %v", err)
			}

			if result.Status != tt.expectStatus {
				t.Fatalf("status = %q, want %q", result.Status, tt.expectStatus)
			}
			if result.SessionID != "sess_123" {
				t.Fatalf("session_id = %q, want %q", result.SessionID, "sess_123")
			}

			rec, ok := service.Subscriber(result.SubscriberID)
			if !ok {
				t.Fatalf("subscriber %q not found", result.SubscriberID)
			}
			if rec.Active != tt.expectActive {
				t.Fatalf("active = %v, want %v", rec.Active, tt.expectActive)
			}
			if rec.Status != tt.expectStatus {
				t.Fatalf("record status = %q, want %q", rec.Status, tt.expectStatus)
			}
			if fake.verifyCalls != 1 {
				t.Fatalf("verify calls = %d, want 1", fake.verifyCalls)
			}
			if fake.smsCalls != 1 {
				t.Fatalf("sms calls = %d, want 1", fake.smsCalls)
			}
			if fake.lastSMS.To != tt.expectSMSTo {
				t.Fatalf("sms to = %q, want %q", fake.lastSMS.To, tt.expectSMSTo)
			}
		})
	}
}
