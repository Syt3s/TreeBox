package dbutil

var (
	// 默认每页 20 条，兼顾常规列表和查询成本。
	DefaultPageSize = 20
	// 游标分页的单页上限，避免一次取太多数据。
	MaxDefaultPageSize = 100
)

type Pagination struct {
	Page     int
	PageSize int
}

func (p Pagination) LimitOffset() (limit, offset int) {
	page := p.Page
	pageSize := p.PageSize

	// 页码非法时回退到第一页。
	if page <= 0 {
		page = 1
	}
	// 每页条数未传时使用默认值。
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	// 返回数据库查询需要的 limit 和 offset。
	return pageSize, (page - 1) * pageSize
}

type Cursor struct {
	Value    interface{}
	PageSize int
}

func (p Cursor) Limit() int {
	pageSize := p.PageSize
	// 未指定时走默认分页大小。
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	// 超出上限时强制收敛，避免大批量扫描。
	if pageSize > MaxDefaultPageSize {
		pageSize = MaxDefaultPageSize
	}
	return pageSize
}
