package main

import (
	"net/http"

	"github.com/Zevyron/gopher-social/internal/store"
)

func (app *application) getUserFeedHandler(w http.ResponseWriter, r *http.Request) {
	// pagination, filters, sort

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

	// TODO: get userID from auth
	feed, err := app.store.Posts.GetUserFeed(ctx, int64(43898), fq)
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

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}