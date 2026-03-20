package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/avinashtandon/business-tracker-backend/internal/service"
	"github.com/avinashtandon/business-tracker-backend/pkg/response"
	"github.com/avinashtandon/business-tracker-backend/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TradeHandler struct {
	tradeSvc service.TradeService
}

func NewTradeHandler(tradeSvc service.TradeService) *TradeHandler {
	return &TradeHandler{tradeSvc: tradeSvc}
}

func (h *TradeHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	var input service.CreateTradeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}
	if err := validator.Validate(input); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	trade, err := h.tradeSvc.CreateTrade(r.Context(), userID, input)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, trade)
}

func (h *TradeHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	trades, err := h.tradeSvc.ListTrades(r.Context(), userID)
	if err != nil {
		response.InternalServerError(w)
		return
	}

	response.Success(w, http.StatusOK, trades)
}

func (h *TradeHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	tradeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.ValidationError(w, "invalid trade id format")
		return
	}

	var input service.CreateTradeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}
	if err := validator.Validate(input); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	err = h.tradeSvc.UpdateTrade(r.Context(), userID, tradeID, input)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "trade not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, http.StatusOK, map[string]string{"message": "trade updated"})
}

func (h *TradeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	tradeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid trade id")
		return
	}

	err = h.tradeSvc.DeleteTrade(r.Context(), userID, tradeID)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "trade not found")
		return
	}
	if err != nil {
		response.InternalServerError(w)
		return
	}

	response.Success(w, http.StatusOK, map[string]string{"message": "trade deleted"})
}
