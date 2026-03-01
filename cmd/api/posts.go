package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/Zevyron/gopher-social/internal/store"
	"github.com/go-chi/chi/v5"
)

type postKey string
const postCtx postKey = "post"

type CreatePostPayload struct {
	Title 		string		`json:"title" validate:"required,max=100"`
	Content		string		`json:"content" validate:"required,max=1000"`
	Tags		[]string	`json:"tags"`
}

type UpdatePostPayload struct {
	Title *string	`json:"title" validate:"omitempty,max=100"`
	Content *string	`json:"content" validate:"omitempty,max=1000"`
}

// createPostHandler godoc
//
//	@Summary		Create a post
//	@Description	Creates a new post authored by the authenticated user
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		CreatePostPayload	true	"Create post payload"
//	@Success		201		{object}	store.Post			"Successfully created post"
//	@Failure		400		{object}	map[string]any		"Invalid request payload"
//	@Failure		500		{object}	map[string]any		"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/posts [post]
func (app *application) createPostHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreatePostPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.validator.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	post := &store.Post{
		Title:   payload.Title,
		Content: payload.Content,
		Tags:    payload.Tags,
		// TODO: Change after auth
		UserID: 1,
	}

	ctx := r.Context()

	if err := app.store.Posts.Create(ctx, post); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, post); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// getPostHandler godoc
//
//	@Summary		Get a post by ID
//	@Description	Fetches a single post along with associated comments
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Param			postID	path		int				true	"Post ID"
//	@Success		200		{object}	store.Post		"Post details with comments"
//	@Failure		400		{object}	map[string]any	"Invalid post ID format"
//	@Failure		404		{object}	map[string]any	"Post not found"
//	@Failure		500		{object}	map[string]any	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/posts/{postID} [get]
func (app *application) getPostHandler(w http.ResponseWriter, r *http.Request) {
	post := getPostFromCtx(r)

	comments, err := app.store.Comments.GetByPostID(r.Context(), post.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	post.Comments = comments

	if err := app.jsonResponse(w, http.StatusOK, post); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// deletePostHandler godoc
//
//	@Summary		Delete a post
//	@Description	Deletes a post by its unique identifier
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Param			postID	path		int				true	"Post ID"
//	@Success		200		{string}	string			"post deleted successfully"
//	@Failure		400		{object}	map[string]any	"Invalid post ID format"
//	@Failure		404		{object}	map[string]any	"Post not found"
//	@Failure		500		{object}	map[string]any	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/posts/{postID} [delete]
func (app *application) deletePostHandler(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "postID")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()

	err = app.store.Posts.DeleteByPostID(ctx, id)
	if err != nil {
		app.notFoundResponse(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, "post deleted successfully"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// updatePostHandler godoc
//
//	@Summary		Update a post
//	@Description	Partially updates a post's title or content by ID
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Param			postID	path		int					true	"Post ID"
//	@Param			payload	body		UpdatePostPayload	true	"Post update payload"
//	@Success		200		{object}	store.Post			"Updated post object"
//	@Failure		400		{object}	map[string]any		"Invalid ID format or invalid JSON payload"
//	@Failure		404		{object}	map[string]any		"Post not found"
//	@Failure		500		{object}	map[string]any		"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/posts/{postID} [patch]
func (app *application) updatePostHandler(w http.ResponseWriter, r *http.Request) {
	post := getPostFromCtx(r)

	var payload UpdatePostPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.validator.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if payload.Content != nil {
		post.Content = *payload.Content
	}

	if payload.Title != nil {
		post.Title = *payload.Title
	}

	if err := app.store.Posts.Update(r.Context(), post); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, post); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) postsContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idParam := chi.URLParam(r, "postID")
		id, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}

		ctx := r.Context()

		post, err := app.store.Posts.GetByID(ctx, id)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrNotFound):
				app.notFoundResponse(w, r, err)
			default:
				app.internalServerError(w, r, err)
			}
			return
		}

		ctx = context.WithValue(ctx, postCtx, post)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getPostFromCtx(r *http.Request) *store.Post {
	post, _ := r.Context().Value(postCtx).(*store.Post)
	return post
}