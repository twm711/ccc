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

// ParseLimitOffset handles the legacy `?limit=&offset=` query style. Clamps
// limit into [1, max] with the provided default when missing or out of range;
// negative offsets become 0. Keeps the existing handler contract — the caller
// stays in charge of passing the (limit, offset) tuple to the repo — but
// enforces a single upper bound so callers cannot ask for `?limit=10000000`.
func ParseLimitOffset(r *http.Request, dflt, max int) (limit, offset int) {
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = dflt
	}
	if limit > max {
		limit = max
	}
	offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
