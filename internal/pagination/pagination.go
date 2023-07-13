package pagination

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/openinfradev/tks-api/internal/helper"
)

const SORT_COLUMN = "sortColumn"
const SORT_ORDER = "sortOrder"
const PAGE_NUMBER = "pageNumber"
const PAGE_SIZE = "pageSize"

type Pagination struct {
	Limit      int
	Page       int
	SortColumn string
	SortOrder  string
	Filters    []Filter
	TotalRows  int64
	TotalPages int
}

type Filter struct {
	Column string
	Values []string
}

var DEFAULT_LIMIT = 10
var MAX_LIMIT = 1000

func (p *Pagination) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}

func (p *Pagination) GetLimit() int {
	if p.Limit == 0 {
		p.Limit = DEFAULT_LIMIT
	}
	return p.Limit
}

func (p *Pagination) GetPage() int {
	if p.Page == 0 {
		p.Page = 1
	}
	return p.Page
}

func (p *Pagination) GetSortColumn() string {
	return p.SortColumn
}

func (p *Pagination) GetSortOrder() string {
	return p.SortOrder
}

func (p *Pagination) GetFilters() []Filter {
	return p.Filters
}

func NewPagination(urlParams *url.Values) *Pagination {
	pg := NewDefaultPagination()

	for key, value := range *urlParams {
		switch key {
		case SORT_COLUMN:
			if value[0] == "" {
				pg.SortColumn = "created_at"
			} else {
				pg.SortColumn = value[0]
			}
		case SORT_ORDER:
			if value[0] == "" {
				pg.SortOrder = "ASC"
			} else {
				pg.SortOrder = value[0]
			}
		case PAGE_NUMBER:
			if value[0] == "" {
				pg.Page = 1
			} else {
				pg.Page, _ = strconv.Atoi(value[0])
			}
		case PAGE_SIZE:
			if value[0] == "" {
				pg.Page = DEFAULT_LIMIT
			} else {
				if limitNum, err := strconv.Atoi(value[0]); err == nil {
					pg.Limit = limitNum
				}
			}
		default:
			pg.Filters = append(pg.Filters, Filter{
				Column: helper.ToSnakeCase(strings.Replace(key, "[]", "", -1)),
				Values: value,
			})
		}
	}

	return pg
}

func NewDefaultPagination() *Pagination {
	return &Pagination{
		SortColumn: "created_at",
		SortOrder:  "ASC",
		Page:       1,
		Limit:      MAX_LIMIT,
	}
}
