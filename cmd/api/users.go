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

func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	if err := app.jsonResponse(w, http.StatusOK, user); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

type FollowUserPayload struct {
		FollowerID int64 `json:"user_id"`
	}

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

		next.ServeHTTP(w,r.WithContext(ctx))
		})
}

func getUserFromContext(r *http.Request) *store.User {
	user, _ := r.Context().Value(userCtx).(*store.User)
	return user
}