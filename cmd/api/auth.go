package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Zevyron/gopher-social/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

type RegisterUserPayload struct {
	FirstName string `json:"first_name" validate:"required,max=100" example:"John"`
	LastName  string `json:"last_name" validate:"required,max=100" example:"Doe"`
	Username  string `json:"username" validate:"required,max=100" example:"johndoe"`
	Email     string `json:"email" validate:"required,email,max=255" example:"john.doe@example.com"`
	Password  string `json:"password" validate:"required,min=3,max=72" example:"supersecret123"`
}

type ResendTokenPayload struct {
	Email string `json:"email" validate:"required,email,max=255" example:"john.doe@example.com"`
}

type CreateUserTokenPayload struct {
	Email string `json:"email" validate:"required,email,max=255"`
	Password	string	`json:"password" validate:"required,min=8,max=72"`
}

// registerUserHandler godoc
//
//	@Summary		Register a user
//	@Description	Registers a new user and sends an activation email with a verification token
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		RegisterUserPayload	true	"User registration credentials"
//	@Success		201		{object}	store.User			"User registered successfully"
//	@Failure		400		{object}	map[string]any		"Invalid payload or validation error"
//	@Failure		409		{object}	map[string]any		"Username or email already exists"
//	@Failure		500		{object}	map[string]any		"Internal server error"
//	@Router			/authentication/user [post]
func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegisterUserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.validator.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &store.User{
		Username: payload.Username,
		FirstName: payload.FirstName,
		LastName: payload.LastName,
		Email: payload.Email,
	}

	

	// hash the user password
	if err := user.Password.Set(payload.Password); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	ctx := r.Context()

	plainToken, tokenHash := store.GenerateInvitationToken() 

	// Commit the user, invitation hash and outbox event in one atomic transaction
	err := app.store.Users.CreateAndInvite(ctx, user, plainToken, tokenHash, app.config.mail.exp)
	if err != nil {
		switch  {
		case errors.Is(err, store.ErrDuplicateEmail), errors.Is(err, store.ErrDuplicateUsername):
			app.conflictResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, nil); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// activateUserHandler godoc
//
//	@Summary		Activate user account
//	@Description	Validates the activation token, marks the user active, and invalidates the token
//	@Tags			authentication
//	@Param			token	path		string	true	"Activation Token"
//	@Success		204		"User activated successfully"
//	@Failure		400		{object}	map[string]any	"Invalid or expired token"
//	@Failure		500		{object}	map[string]any	"Internal server error"
//	@Router			/authentication/activate/{token} [put]
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		app.badRequestResponse(w, r, errors.New("activation token is required"))
		return
	}

	ctx := r.Context()
	err := app.store.Users.Activate(ctx, token)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrInvalidOrExpiredToken):
			app.badRequestResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.jsonResponse(w, http.StatusNoContent, nil); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// resendActivationTokenHandler godoc
//
//	@Summary		Resend activation token
//	@Description	Generates and emails a new activation token for an unactivated account
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		ResendTokenPayload	true	"User email"
//	@Success		202		{object}	map[string]string	"Request accepted"
//	@Failure		400		{object}	map[string]any		"Invalid payload"
//	@Failure		500		{object}	map[string]any		"Internal server error"
//	@Router			/authentication/token/resend [post]
func (app *application) resendActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var payload ResendTokenPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.validator.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	plainToken, tokenHash := store.GenerateInvitationToken()
	ctx := r.Context()

	err := app.store.Users.ResendInvitation(ctx, payload.Email, plainToken, tokenHash, app.config.mail.exp)
	if err != nil && !errors.Is(err, store.ErrUserNotFoundOrActive) {
		app.internalServerError(w, r, err)
		return
	}

	// Always return 202 Accepted to prevent account enumeration
	response := map[string]string{
		"message": "If an unactivated account with that email exists, a fresh activation link has been sent.",
	}

	if err := app.jsonResponse(w, http.StatusAccepted, response); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) CreateTokenHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateUserTokenPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.validator.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()
	user, err := app.store.Users.GetByEmail(ctx, payload.Email)
	fmt.Println(err)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.unauthorizedResponse(w, r, errors.New("invalid credentials"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if !user.IsActive {
		app.unauthorizedResponse(w, r, errors.New("account is not activated, please check your email"))
		return
	}

	matched, err := user.Password.Matches(payload.Password)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if !matched {
		app.unauthorizedResponse(w, r, errors.New("invalid credentials"))
		return
	}

	claims := jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(app.config.auth.token.exp).Unix(),
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(),
		"iss": app.config.auth.token.iss,
		"aud": app.config.auth.token.iss,
	}

	token, err := app.authenticator.GenerateToken(claims)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	response := map[string]any{
		"token": token,
	}

	if err := app.jsonResponse(w, http.StatusCreated, response); err != nil {
		app.internalServerError(w, r, err)
	}
}