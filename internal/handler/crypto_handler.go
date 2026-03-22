package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/avinashtandon/business-tracker-backend/internal/dto"
	"github.com/avinashtandon/business-tracker-backend/internal/middleware"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/avinashtandon/business-tracker-backend/internal/service"
	"github.com/avinashtandon/business-tracker-backend/pkg/coingecko"
	"github.com/avinashtandon/business-tracker-backend/pkg/response"
	"github.com/avinashtandon/business-tracker-backend/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CryptoHandler struct {
	cryptoSvc service.CryptoService
}

func NewCryptoHandler(cryptoSvc service.CryptoService) *CryptoHandler {
	return &CryptoHandler{cryptoSvc: cryptoSvc}
}

// POST /api/v1/crypto
func (h *CryptoHandler) CreateHolding(w http.ResponseWriter, r *http.Request) {
	userID := middleware.MustUserID(r.Context())

	var input dto.CreateHoldingRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}
	if err := validator.Validate(input); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	holdingResp, err := h.cryptoSvc.CreateHolding(r.Context(), userID, input)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, holdingResp)
}

// GET /api/v1/crypto
func (h *CryptoHandler) ListHoldings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.MustUserID(r.Context())

	holdingsResp, err := h.cryptoSvc.ListHoldings(r.Context(), userID)
	if err != nil {
		response.InternalServerError(w)
		return
	}

	response.Success(w, http.StatusOK, holdingsResp)
}

// DELETE /api/v1/crypto/:id
func (h *CryptoHandler) DeleteHolding(w http.ResponseWriter, r *http.Request) {
	userID := middleware.MustUserID(r.Context())

	holdingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.ValidationError(w, "invalid holding id format")
		return
	}

	err = h.cryptoSvc.DeleteHolding(r.Context(), userID, holdingID)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "holding not found")
		return
	}
	if err != nil {
		response.InternalServerError(w)
		return
	}

	response.Success(w, http.StatusOK, map[string]string{"message": "holding deleted"})
}

// POST /api/v1/crypto/:id/purchases
func (h *CryptoHandler) CreatePurchase(w http.ResponseWriter, r *http.Request) {
	userID := middleware.MustUserID(r.Context())

	holdingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.ValidationError(w, "invalid holding id format")
		return
	}

	var input dto.CreatePurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}
	if err := validator.Validate(input); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	purchaseResp, err := h.cryptoSvc.CreatePurchase(r.Context(), userID, holdingID, input)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "holding not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, purchaseResp)
}

// DELETE /api/v1/crypto/:id/purchases/:pid
func (h *CryptoHandler) DeletePurchase(w http.ResponseWriter, r *http.Request) {
	userID := middleware.MustUserID(r.Context())

	holdingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.ValidationError(w, "invalid holding id format")
		return
	}

	purchaseID, err := uuid.Parse(chi.URLParam(r, "pid"))
	if err != nil {
		response.ValidationError(w, "invalid purchase id format")
		return
	}

	err = h.cryptoSvc.DeletePurchase(r.Context(), userID, holdingID, purchaseID)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "purchase or holding not found")
		return
	}
	if err != nil {
		response.InternalServerError(w)
		return
	}

	response.Success(w, http.StatusOK, map[string]string{"message": "purchase deleted"})
}

// GET /api/v1/crypto/prices?symbols=BTC,ETH,SOL
func (h *CryptoHandler) GetPrices(w http.ResponseWriter, r *http.Request) {
	symbolsParam := strings.TrimSpace(r.URL.Query().Get("symbols"))
	if symbolsParam == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"symbols query param is required (e.g. ?symbols=BTC,ETH)")
		return
	}

	symbols := strings.Split(symbolsParam, ",")
	for i, s := range symbols {
		symbols[i] = strings.TrimSpace(strings.ToUpper(s))
	}

	result, err := coingecko.Cache.GetPrices(symbols)
	if err != nil {
		response.Error(w, http.StatusBadGateway, "PRICE_FETCH_FAILED",
			"Failed to fetch prices from CoinGecko")
		return
	}

	response.Success(w, http.StatusOK, result)
}
