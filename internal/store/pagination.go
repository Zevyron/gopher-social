package store

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PaginatedFeedQuery struct {
	Limit  	int    		`json:"limit" validate:"gte=1,lte=20"`
	Sort   	string 		`json:"sort" validate:"oneof=asc desc"`
	Cursor 	string 		`json:"cursor"`
	Tags	[]string	`json:"tags" validate:"max=30"`
	Search	string		`json:"search" validate:"max=100"`
	Since	string 		`json:"since"`
	Until	string		`json:"until"`
}

func (fq PaginatedFeedQuery) Parse(r *http.Request) (PaginatedFeedQuery, error) {
	// pagiantion, filters, sort
	qs := r.URL.Query()

	limit := qs.Get("limit")
	if limit != "" {
		l, err := strconv.Atoi(limit)
		if err != nil {
			return fq, err
		}

		fq.Limit = l
	}

	sort := qs.Get("sort")
	if sort != "" {
		fq.Sort = sort
	}

	cursor := qs.Get("cursor")
	if cursor != "" {
		fq.Cursor = cursor
	}

	tags := qs.Get("tags")
	if tags != "" {
		fq.Tags = strings.Split(tags, ",")
	}

	search := qs.Get("search")
	if search != "" {
		fq.Search = search
	}

	// RFC3339 includes hours, minutes, seconds and a time zone offset
	if since := qs.Get("since"); since != "" {
		if _, err := time.Parse(time.RFC3339, since); err != nil {
			return fq, fmt.Errorf("invalid 'since' format, expected RFC3339 (e.g., 2026-01-01T00:00:00Z)")
		}
		fq.Since = since
	}

	if until := qs.Get("until"); until != "" {
		if _, err := time.Parse(time.RFC3339, until); err != nil {
			return fq, fmt.Errorf("invalid 'until' format, expected RFC3339 (e.g., 2026-03-01T00:00:00Z)")
		}
		fq.Until = until
	}

	return fq, nil
}

// GenerateCursor creates a URL-safe Base64 string from the last post's timestamp and ID
func GenerateCursor(createdAt string, id int64) string {
	raw := fmt.Sprintf("%s,%d", createdAt, id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes the URL-safe Base64 string back into a timestamp and ID for SQL
func DecodeCursor(encoded string) (string, int64, error) {
	if encoded == "" {
		return "9999-12-31 23:59:59", 9223372036854775807, nil
	}

	decodedBytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", 0, fmt.Errorf("base64 decode failed: %w", err)
	}

	decoded := string(decodedBytes)
	parts := strings.Split(decoded, ",")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid cursor format")
	}

	createdAt := parts[0]
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid cursor id: %w", err)
	}

	return createdAt, id, nil
}