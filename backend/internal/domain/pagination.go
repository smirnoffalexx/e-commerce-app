package domain

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	MaxPage         = 10000
)

type PageRequest struct {
	Page     int
	PageSize int
}

func (r PageRequest) Normalize() (PageRequest, error) {
	if r.Page == 0 {
		r.Page = 1
	}
	if r.PageSize == 0 {
		r.PageSize = DefaultPageSize
	}

	var verr ValidationError
	if r.Page < 1 || r.Page > MaxPage {
		verr.Add("page", CodeInvalidFormat, "page must be between 1 and 10000")
	}
	if r.PageSize < 1 || r.PageSize > MaxPageSize {
		verr.Add("page_size", CodeInvalidFormat, "page_size must be between 1 and 100")
	}
	return r, verr.Err()
}

func (r PageRequest) Offset() int {
	return (r.Page - 1) * r.PageSize
}

type Page[T any] struct {
	Items    []T
	Total    int64
	Page     int
	PageSize int
}

func (p Page[T]) TotalPages() int {
	if p.Total == 0 {
		return 0
	}
	return int((p.Total + int64(p.PageSize) - 1) / int64(p.PageSize))
}
