package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/openinfradev/tks-api/internal/usecase"
	"github.com/openinfradev/tks-api/pkg/domain"
)

type AuthHandler struct {
	usecase usecase.IAuthUsecase
}

func NewAuthHandler(h usecase.IAuthUsecase) *AuthHandler {
	return &AuthHandler{
		usecase: h,
	}
}

// Login godoc
// @Tags Auth
// @Summary login
// @Description login
// @Accept json
// @Produce json
// @Param body body domain.LoginRequest true "account info"
// @Success 200 {object} domain.User "user detail"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	input := domain.LoginRequest{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrorJSON(w, fmt.Sprintf("Invalid request. %s", err), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &input)
	if err != nil {
		ErrorJSON(w, fmt.Sprintf("Invalid request. %s", err), http.StatusBadRequest)
		return
	}

	user, err := h.usecase.Login(input.AccountId, input.Password)
	if err != nil {
		InternalServerError(w, err)
		return
	}

	var out struct {
		User domain.User `json:"user"`
	}

	out.User = user

	//_ = h.Repository.AddHistory(user.ID.String(), "", "login", fmt.Sprintf("[%s] 님이 로그인하였습니다.", input.AccountId))

	ResponseJSON(w, out, "", http.StatusOK)

}

// Signup godoc
// @Tags Auth
// @Summary signup
// @Description signup
// @Accept json
// @Produce json
// @Param body body domain.SignUpRequest true "account info"
// @Success 200 {object} domain.User
// @Router /auth/signup [post]
// @Security     JWT
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	input := domain.SignUpRequest{}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrorJSON(w, fmt.Sprintf("Invalid request. %s", err), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &input)
	if err != nil {
		ErrorJSON(w, fmt.Sprintf("Invalid request. %s", err), http.StatusBadRequest)
		return
	}

	user, err := h.usecase.Register(input.AccountId, input.Password, input.Name)
	if err != nil {
		InternalServerError(w, err)
		return
	}

	var out struct {
		User domain.User
	}

	out.User = user

	ResponseJSON(w, out, "", http.StatusOK)
}
