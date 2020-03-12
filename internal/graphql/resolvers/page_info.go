package resolvers

import "fmt"

// PageInfo represents general resolvable information about the current page of a list of elements.
type ListPageInfo struct {
	First       *Cursor
	Last        *Cursor
	HasNext     bool
	HasPrevious bool
}

// NewListPageInfo creates a new page information structure.
func NewListPageInfo(first *Cursor, last *Cursor, hasNext bool, hasPrevious bool) (*ListPageInfo, error) {
	// make sure cursors are given
	if first == nil || last == nil {
		return nil, fmt.Errorf("missing one of the cursors")
	}

	// make the structure
	return &ListPageInfo{
		First:       first,
		Last:        last,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
	}, nil
}