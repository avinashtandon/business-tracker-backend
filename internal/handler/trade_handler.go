package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/dto"
	"github.com/avinashtandon/business-tracker-backend/internal/middleware"
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
	userID := middleware.MustUserID(r.Context())

	var input dto.CreateTradeRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}
	if err := validator.Validate(input); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	tradeResp, err := h.tradeSvc.CreateTrade(r.Context(), userID, input)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, tradeResp)
}

func (h *TradeHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.MustUserID(r.Context())

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	cursorStr := r.URL.Query().Get("cursor")
	var cursor *time.Time
	if cursorStr != "" {
		cursorTime, err := time.Parse(time.RFC3339Nano, cursorStr)
		if err == nil {
			cursor = &cursorTime
		}
	}

	trades, hasMore, err := h.tradeSvc.ListTrades(r.Context(), userID, cursor, limit)
	if err != nil {
		response.InternalServerError(w)
		return
	}

	var nextCursor *string
	if hasMore && len(trades) > 0 {
		c := trades[len(trades)-1].CreatedAt
		nextCursor = &c
	}

	meta := map[string]interface{}{
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	}

	response.SuccessWithMeta(w, http.StatusOK, trades, meta)
}

func (h *TradeHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.MustUserID(r.Context())

	tradeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.ValidationError(w, "invalid trade id format")
		return
	}

	var input dto.CreateTradeRequest
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
	userID := middleware.MustUserID(r.Context())

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
