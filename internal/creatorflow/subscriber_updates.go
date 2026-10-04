package creatorflow

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sync"
)

type SignupIntent struct {
	Phone     string `json:"phone"`
	Campaign  string `json:"campaign"`
	Locale    string `json:"locale"`
	AssetID   string `json:"asset_id"`
	AssetURL  string `json:"asset_url"`
	BuyerName string `json:"buyer_name"`
}

type VerifyPurchaseInput struct {
	Phone     string `json:"phone"`
	Code      string `json:"code"`
	Campaign  string `json:"campaign"`
	AssetID   string `json:"asset_id"`
	AssetURL  string `json:"asset_url"`
	BuyerName string `json:"buyer_name"`
	Login     bool   `json:"login"`
}

type SubscriberRecord struct {
	SubscriberID string `json:"subscriber_id"`
	UserID       string `json:"user_id"`
	Phone        string `json:"phone"`
	Campaign     string `json:"campaign"`
	AssetID      string `json:"asset_id"`
	Status       string `json:"status"`
	Active       bool   `json:"active"`
}

type DeliveryResult struct {
	SubscriberID string `json:"subscriber_id"`
	Status       string `json:"status"`
	DownloadURL  string `json:"download_url"`
	SessionID    string `json:"session_id"`
}

type SMSGateway interface {
	AuthPhoneSendCode(ctx context.Context, reqBody AuthPhoneSendCodeRequest) error
	AuthPhoneVerify(ctx context.Context, reqBody AuthPhoneVerifyRequest) error
	AuthIdentityResolve(ctx context.Context, reqBody AuthIdentityResolveRequest) (AuthIdentityResolveResponse, error)
	AuthSessionCreate(ctx context.Context, reqBody AuthSessionCreateRequest) (AuthSessionCreateResponse, error)
	SMSOTP(ctx context.Context, reqBody SMSOTPRequest) error
}

type Service struct {
	infrai      SMSGateway
	mu          sync.Mutex
	subscribers map[string]SubscriberRecord
}

func NewService(infrai SMSGateway) *Service {
	return &Service{
		infrai:      infrai,
		subscribers: map[string]SubscriberRecord{},
	}
}

func (s *Service) RequestSignupCode(ctx context.Context, in SignupIntent) error {
	return s.infrai.AuthPhoneSendCode(ctx, AuthPhoneSendCodeRequest{
		Phone:   in.Phone,
		Purpose: "login",
		Locale:  in.Locale,
	})
}

func (s *Service) ActivatePurchase(ctx context.Context, in VerifyPurchaseInput) (DeliveryResult, error) {
	if err := s.infrai.AuthPhoneVerify(ctx, AuthPhoneVerifyRequest{
		Phone: in.Phone,
		Code:  in.Code,
		Login: in.Login,
	}); err != nil {
		return DeliveryResult{}, err
	}

	identity, err := s.infrai.AuthIdentityResolve(ctx, AuthIdentityResolveRequest{
		Type:  "phone",
		Value: in.Phone,
	})
	if err != nil {
		return DeliveryResult{}, err
	}

	session, err := s.infrai.AuthSessionCreate(ctx, AuthSessionCreateRequest{
		UserID: identity.UserID,
		Method: "phone",
	})
	if err != nil {
		return DeliveryResult{}, err
	}

	record := SubscriberRecord{
		SubscriberID: subscriberID(in.Phone, in.AssetID),
		UserID:       identity.UserID,
		Phone:        in.Phone,
		Campaign:     in.Campaign,
		AssetID:      in.AssetID,
		Status:       "delivered",
		Active:       true,
	}

	s.mu.Lock()
	s.subscribers[record.SubscriberID] = record
	s.mu.Unlock()

	if err := s.infrai.SMSOTP(ctx, SMSOTPRequest{
		To: in.Phone,
	}); err != nil {
		return DeliveryResult{}, err
	}

	return DeliveryResult{
		SubscriberID: record.SubscriberID,
		Status:       record.Status,
		DownloadURL:  in.AssetURL,
		SessionID:    session.SessionID,
	}, nil
}

func (s *Service) Subscriber(subscriberID string) (SubscriberRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.subscribers[subscriberID]
	return rec, ok
}

func subscriberID(phone, assetID string) string {
	h := sha1.Sum([]byte(phone + ":" + assetID))
	return "sub_" + hex.EncodeToString(h[:8])
}
