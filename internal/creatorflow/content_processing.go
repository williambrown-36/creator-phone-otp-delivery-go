package creatorflow

import (
	"encoding/json"
	"net/http"
)

type requestCodeEnvelope struct {
	Phone     string `json:"phone"`
	Campaign  string `json:"campaign"`
	Locale    string `json:"locale"`
	AssetID   string `json:"asset_id"`
	AssetURL  string `json:"asset_url"`
	BuyerName string `json:"buyer_name"`
}

type verifyAndDeliverEnvelope struct {
	Phone     string `json:"phone"`
	Code      string `json:"code"`
	Campaign  string `json:"campaign"`
	AssetID   string `json:"asset_id"`
	AssetURL  string `json:"asset_url"`
	BuyerName string `json:"buyer_name"`
	Login     bool   `json:"login"`
}

func RequestCodeHandler(service *Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req requestCodeEnvelope
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := service.RequestSignupCode(r.Context(), SignupIntent{
			Phone:     req.Phone,
			Campaign:  req.Campaign,
			Locale:    req.Locale,
			AssetID:   req.AssetID,
			AssetURL:  req.AssetURL,
			BuyerName: req.BuyerName,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "code_sent",
		})
	})
}

func VerifyAndDeliverHandler(service *Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req verifyAndDeliverEnvelope
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := service.ActivatePurchase(r.Context(), VerifyPurchaseInput{
			Phone:     req.Phone,
			Code:      req.Code,
			Campaign:  req.Campaign,
			AssetID:   req.AssetID,
			AssetURL:  req.AssetURL,
			BuyerName: req.BuyerName,
			Login:     req.Login,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
}
