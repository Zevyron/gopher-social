package main

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Zevyron/gopher-social/internal/store"
	"github.com/go-chi/chi/v5"
)

type userKey string
const userCtx userKey = "user"

type FollowUserPayload struct {
	FollowerID int64 `json:"user_id"`
}

// getUserHandler godoc
//
//	@Summary		Get a user profile
//	@Description	Fetches a user profile by ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userID	path		int				true	"User ID"
//	@Success		200		{object}	store.User		"User profile details"
//	@Failure		400		{object}	map[string]any	"Invalid user ID format or user not found"
//	@Failure		500		{object}	map[string]any	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/users/{userID} [get]
func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	if err := app.jsonResponse(w, http.StatusOK, user); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// followUserHandler godoc
//
//	@Summary		Follow a user
//	@Description	Follows a target user by ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userID	path		int					true	"Target User ID"
//	@Param			payload	body		FollowUserPayload	true	"Follower payload"
//	@Success		204		"User followed successfully"
//	@Failure		400		{object}	map[string]any		"Invalid user ID or wrong request"
//	@Failure		500		{object}	map[string]any		"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/users/{userID}/follow [put]
func (app *application) followUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	userID := user.ID

	// TODO: Revert back to auth followerID from ctx
	var payload FollowUserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	followerID := payload.FollowerID

	ctx := r.Context()

	if err := app.store.Followers.Follow(ctx, userID, followerID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusNoContent, nil); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// unfollowUserHandler godoc
//
//	@Summary		Unfollow a user
//	@Description	Unfollows a target user by ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userID	path		int					true	"Target User ID"
//	@Param			payload	body		FollowUserPayload	true	"Follower payload"
//	@Success		204		"User unfollowed successfully"
//	@Failure		400		{object}	map[string]any		"Invalid user ID or malformed request"
//	@Failure		500		{object}	map[string]any		"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/users/{userID}/unfollow [put]
func (app *application) unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	userID := user.ID

	// TODO: Revert back to auth followerID from ctx
	var payload FollowUserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	followerID := payload.FollowerID

	ctx := r.Context()

	if err := app.store.Followers.Unfollow(ctx, userID, followerID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusNoContent, nil); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) userContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}

		ctx := r.Context()

		user, err := app.store.Users.GetByID(ctx, userID)
		if err != nil {
			switch err {
			case store.ErrNotFound:
				app.badRequestResponse(w, r, err)
				return
			default:
				app.internalServerError(w, r, err)
				return
			}
		}

		ctx = context.WithValue(ctx, userCtx, user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserFromContext(r *http.Request) *store.User {
	user, _ := r.Context().Value(userCtx).(*store.User)
	return user
}