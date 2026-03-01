package store

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type PaginatedFeedQuery struct {
	Limit  int    `json:"limit" validate:"gte=1,lte=20"`
	Sort   string `json:"sort" validate:"oneof=asc desc"`
	Cursor string `json:"cursor"`
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