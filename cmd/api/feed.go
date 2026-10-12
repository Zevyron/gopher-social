package main

import (
	"net/http"

	"github.com/Zevyron/gopher-social/internal/store"
)

// getUserFeedHandler godoc
//
//	@Summary		Fetches the user's feed
//	@Description	Retrieves a cursor-paginated timeline with tags, text search, and RFC3339 date-range filters.
//	@Tags			feed
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int		false	"Items per page (1-20)"						default(20)
//	@Param			sort	query		string	false	"Sort direction (asc, desc)"				default(desc)
//	@Param			cursor	query		string	false	"URL-safe base64 composite cursor"
//	@Param			tags	query		string	false	"Comma-separated tags (e.g., go,backend)"
//	@Param			search	query		string	false	"Search keyword in title or content"
//	@Param			since	query		string	false	"Lower bound timestamp (RFC3339, e.g., 2026-02-01T00:00:00Z)"
//	@Param			until	query		string	false	"Upper bound timestamp (RFC3339, e.g., 2026-02-28T23:59:59Z)"
//	@Success		200		{object}	map[string]any	"Feed items and pagination metadata"
//	@Failure		400		{object}	map[string]any	"Bad request (invalid dates or query params)"
//	@Failure		500		{object}	map[string]any	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/users/feed [get]
func (app *application) getUserFeedHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	fq := store.PaginatedFeedQuery{
		Limit: 	20,
		Sort: 	"desc",
	}

	fq, err := fq.Parse(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.validator.Struct(fq); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()

	feed, err := app.store.Posts.GetUserFeed(ctx, user.ID, fq)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	var nextCursor string
	if len(feed) > 0 {
		lastPost := feed[len(feed)-1]
		nextCursor = store.GenerateCursor(lastPost.CreatedAt, lastPost.ID)
	}

	response := struct {
		Data []store.PostWithMetadata 	`json:"data"`
		NextCursor string				`json:"next_cursor"`
	}{
		Data: feed,
		NextCursor: nextCursor,
	}

	if err := app.jsonResponse(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}