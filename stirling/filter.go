package stirling

import "context"

// FilterResult is the outcome of a filter call. Matched is false when the
// server answers 204 No Content; Result is then nil. When Matched is true,
// Result carries the original PDF and must be closed.
type FilterResult struct {
	Matched bool
	Result  *Result
}

// Close closes the matched document, if any.
func (f *FilterResult) Close() error {
	if f == nil || f.Result == nil {
		return nil
	}
	return f.Result.Close()
}

// FilterPageSizeOptions configures [Client.FilterPageSize].
type FilterPageSizeOptions struct {
	Comparator Comparator
	PageSize   PageSize
}

// FilterPageRotationOptions configures [Client.FilterPageRotation].
type FilterPageRotationOptions struct {
	Comparator Comparator
	Rotation   int
}

// FilterPageCountOptions configures [Client.FilterPageCount].
type FilterPageCountOptions struct {
	Comparator Comparator
	Count      int
}

// FilterFileSizeOptions configures [Client.FilterFileSize]. Size is in bytes.
type FilterFileSizeOptions struct {
	Comparator Comparator
	Size       int64
}

// FilterContainsTextOptions configures [Client.FilterContainsText].
type FilterContainsTextOptions struct {
	Pages string
	Text  string
}

// FilterContainsImageOptions configures [Client.FilterContainsImage].
type FilterContainsImageOptions struct {
	Pages string
}

// FilterPageSize returns the document when its page size satisfies the
// comparison.
func (c *Client) FilterPageSize(ctx context.Context, in FileInput, opts FilterPageSizeOptions) (*FilterResult, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "comparator", Value: string(opts.Comparator)},
		Param{Name: "standardPageSize", Value: string(opts.PageSize)},
	)
	return c.filter(ctx, "/api/v1/filter/filter-page-size", params)
}

// FilterPageRotation returns the document when its rotation satisfies the
// comparison.
func (c *Client) FilterPageRotation(ctx context.Context, in FileInput, opts FilterPageRotationOptions) (*FilterResult, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "comparator", Value: string(opts.Comparator)},
		Param{Name: "rotation", Value: opts.Rotation},
	)
	return c.filter(ctx, "/api/v1/filter/filter-page-rotation", params)
}

// FilterPageCount returns the document when its page count satisfies the
// comparison.
func (c *Client) FilterPageCount(ctx context.Context, in FileInput, opts FilterPageCountOptions) (*FilterResult, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "comparator", Value: string(opts.Comparator)},
		Param{Name: "pageCount", Value: opts.Count},
	)
	return c.filter(ctx, "/api/v1/filter/filter-page-count", params)
}

// FilterFileSize returns the document when its size satisfies the comparison.
func (c *Client) FilterFileSize(ctx context.Context, in FileInput, opts FilterFileSizeOptions) (*FilterResult, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "comparator", Value: string(opts.Comparator)},
		Param{Name: "fileSize", Value: opts.Size},
	)
	return c.filter(ctx, "/api/v1/filter/filter-file-size", params)
}

// FilterContainsText returns the document when the selected pages contain the
// text.
func (c *Client) FilterContainsText(ctx context.Context, in FileInput, opts FilterContainsTextOptions) (*FilterResult, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.Pages != "" {
		params = append(params, Param{Name: "pageNumbers", Value: opts.Pages})
	}
	params = append(params, Param{Name: "text", Value: opts.Text})
	return c.filter(ctx, "/api/v1/filter/filter-contains-text", params)
}

// FilterContainsImage returns the document when the selected pages contain an
// image.
func (c *Client) FilterContainsImage(ctx context.Context, in FileInput, opts FilterContainsImageOptions) (*FilterResult, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.Pages != "" {
		params = append(params, Param{Name: "pageNumbers", Value: opts.Pages})
	}
	return c.filter(ctx, "/api/v1/filter/filter-contains-image", params)
}

func (c *Client) filter(ctx context.Context, path string, params []Param) (*FilterResult, error) {
	res, err := c.Call(ctx, path, params...)
	if err != nil {
		return nil, err
	}
	if res.IsEmpty() {
		return &FilterResult{Matched: false}, nil
	}
	return &FilterResult{Matched: true, Result: res}, nil
}
