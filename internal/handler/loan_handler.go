package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/avinashtandon/business-tracker-backend/internal/middleware"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/avinashtandon/business-tracker-backend/internal/service"
	"github.com/avinashtandon/business-tracker-backend/pkg/response"
	"github.com/avinashtandon/business-tracker-backend/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type LoanHandler struct {
	loanSvc service.LoanService
}

func NewLoanHandler(loanSvc service.LoanService) *LoanHandler {
	return &LoanHandler{loanSvc: loanSvc}
}

func getUserID(r *http.Request) (uuid.UUID, error) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return uuid.Nil, errors.New("unauthorized")
	}
	return uuid.Parse(claims.Subject)
}

func (h *LoanHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	var input service.CreateLoanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}
	if err := validator.Validate(input); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	loan, err := h.loanSvc.CreateLoan(r.Context(), userID, input)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, loan)
}

func (h *LoanHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	loans, err := h.loanSvc.ListLoans(r.Context(), userID)
	if err != nil {
		response.InternalServerError(w)
		return
	}

	response.Success(w, http.StatusOK, loans)
}

func (h *LoanHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	loanID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.ValidationError(w, "invalid loan id format")
		return
	}

	loan, err := h.loanSvc.GetLoan(r.Context(), userID, loanID)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "loan not found")
		return
	}
	if err != nil {
		response.InternalServerError(w)
		return
	}
	response.Success(w, http.StatusOK, loan)
}

func (h *LoanHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	loanID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.ValidationError(w, "invalid loan id format")
		return
	}

	var input service.CreateLoanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}
	if err := validator.Validate(input); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	err = h.loanSvc.UpdateLoan(r.Context(), userID, loanID, input)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "loan not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, http.StatusOK, map[string]string{"message": "loan updated"})
}

func (h *LoanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	loanID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid loan id")
		return
	}

	err = h.loanSvc.DeleteLoan(r.Context(), userID, loanID)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "loan not found")
		return
	}
	if err != nil {
		response.InternalServerError(w)
		return
	}

	response.Success(w, http.StatusOK, map[string]string{"message": "loan deleted"})
}

func (h *LoanHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	loanID, err := uuid.Parse(chi.URLParam(r, "loan_id"))
	if err != nil {
		response.ValidationError(w, "invalid loan id format")
		return
	}

	var input service.CreateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}
	if err := validator.Validate(input); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	tx, err := h.loanSvc.CreateTransaction(r.Context(), userID, loanID, input)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "loan not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, tx)
}

func (h *LoanHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	loanID, err := uuid.Parse(chi.URLParam(r, "loan_id"))
	if err != nil {
		response.ValidationError(w, "invalid loan id format")
		return
	}

	txID, err := uuid.Parse(chi.URLParam(r, "transaction_id"))
	if err != nil {
		response.ValidationError(w, "invalid transaction id format")
		return
	}

	err = h.loanSvc.DeleteTransaction(r.Context(), userID, loanID, txID)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "transaction or loan not found")
		return
	}
	if err != nil {
		response.InternalServerError(w)
		return
	}

	response.Success(w, http.StatusOK, map[string]string{"message": "transaction deleted"})
}
