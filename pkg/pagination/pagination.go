package pagination

import (
	"net/http"
	"strconv"
)

type Params struct {
	Page     int
	PageSize int
	Offset   int
}

func Parse(r *http.Request) Params {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return Params{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}

// CursorParams supports keyset (cursor-based) pagination. The cursor is the
// last-seen ID; results are returned in descending order starting after that ID.
type CursorParams struct {
	Cursor   int64
	PageSize int
}

// ParseCursor extracts cursor-based pagination params from the request.
// Falls back to offset-based Params when no cursor is provided.
func ParseCursor(r *http.Request) CursorParams {
	cursor, _ := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64)
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return CursorParams{Cursor: cursor, PageSize: pageSize}
}
