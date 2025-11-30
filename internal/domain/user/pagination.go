package user

// Pagination represents pagination information for list responses.
type Pagination struct {
	Total      int64 // Total number of records
	Page       int64 // Current page number (1-based)
	Limit      int64 // Number of records per page
	TotalPages int64 // Total number of pages
}

// NewPagination creates a new Pagination instance with calculated total pages.
func NewPagination(total, page, limit int64) *Pagination {
	totalPages := limit
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return &Pagination{
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}
