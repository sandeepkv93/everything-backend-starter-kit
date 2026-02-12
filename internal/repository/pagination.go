package repository

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type PageRequest struct {
	Page     int
	PageSize int
}

type PageResult[T any] struct {
	Items      []T
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
}

func normalizePageRequest(in PageRequest) PageRequest {
	page := in.Page
	if page < 1 {
		page = DefaultPage
	}
	pageSize := in.PageSize
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return PageRequest{Page: page, PageSize: pageSize}
}

func calcTotalPages(total int64, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 0
	}
	ps := int64(pageSize)
	pages := total / ps
	if total%ps != 0 {
		pages++
	}
	maxInt := int64(^uint(0) >> 1)
	if pages > maxInt {
		return int(maxInt)
	}
	return int(pages)
}
