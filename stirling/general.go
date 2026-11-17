package stirling

import (
	"context"
	"fmt"
)

// SplitSectionsOptions configures [Client.SplitBySections]. Each page is cut
// into a grid of horizontal divisions and vertical divisions.
type SplitSectionsOptions struct {
	HorizontalDivisions int
	VerticalDivisions   int
	Merge               bool
}

// SplitChaptersOptions configures [Client.SplitByChapters].
type SplitChaptersOptions struct {
	IncludeMetadata *bool
	AllowDuplicates *bool
	BookmarkLevel   *int
}

// SplitSizeOptions configures [Client.SplitBySizeOrCount]. SplitValue is a
// size such as "10MB" for [SplitBySize] or a count such as "5" for the count
// modes.
type SplitSizeOptions struct {
	SplitType  SplitType
	SplitValue string
}

// ScaleOptions configures [Client.ScalePages].
type ScaleOptions struct {
	PageSize    PageSize
	ScaleFactor float64
}

// RearrangeOptions configures [Client.RearrangePages]. Pages is the new page
// order for [RearrangeCustom], for example "3,1,2,4".
type RearrangeOptions struct {
	Pages string
	Mode  RearrangeMode
}

// OverlayOptions configures [Client.OverlayPDFs]. Counts is only used with
// [OverlayFixedRepeat] and must match the number of overlay files.
type OverlayOptions struct {
	Mode     OverlayMode
	Position OverlayPosition
	Counts   []int
}

// MultiPageOptions configures [Client.MultiPageLayout].
type MultiPageOptions struct {
	PagesPerSheet int
	AddBorder     bool
}

// MergeOptions configures [Client.Merge]. Nil fields are left to the server,
// whose defaults are "orderProvided", removeCertSign=true, and
// generateToc=false.
type MergeOptions struct {
	SortType       SortType
	RemoveCertSign *bool
	GenerateTOC    *bool
}

// CropOptions configures [Client.Crop], in PDF points.
type CropOptions struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// BookletOptions configures [Client.Booklet].
type BookletOptions struct {
	Type          BookletType
	PagesPerSheet int
	AddBorder     bool
	Orientation   Orientation
}

// SplitBySections cuts every page into a grid of sections. The server returns
// a PDF when Merge is true and a ZIP archive otherwise.
func (c *Client) SplitBySections(ctx context.Context, in FileInput, opts SplitSectionsOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "horizontalDivisions", Value: opts.HorizontalDivisions},
		Param{Name: "verticalDivisions", Value: opts.VerticalDivisions},
		Param{Name: "merge", Value: opts.Merge},
	)
	return c.Call(ctx, "/api/v1/general/split-pdf-by-sections", params...)
}

// SplitByChapters splits a document at each bookmark. The result is a ZIP
// archive with one PDF per chapter.
func (c *Client) SplitByChapters(ctx context.Context, in FileInput, opts SplitChaptersOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "includeMetadata", Value: opts.IncludeMetadata},
		Param{Name: "allowDuplicates", Value: opts.AllowDuplicates},
		Param{Name: "bookmarkLevel", Value: opts.BookmarkLevel},
	)
	return c.Call(ctx, "/api/v1/general/split-pdf-by-chapters", params...)
}

// SplitPages splits a PDF after the given page numbers and returns a ZIP
// archive. Page numbers use Stirling's syntax: "1,3,5-9", "all", or an
// expression such as "2n+1".
func (c *Client) SplitPages(ctx context.Context, in FileInput, pages string) (*Result, error) {
	if pages == "" {
		return nil, fmt.Errorf("stirling: split pages: page numbers are required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "pageNumbers", Value: pages})
	return c.Call(ctx, "/api/v1/general/split-pages", params...)
}

// SplitBySizeOrCount splits a PDF into several documents by size, page count,
// or document count. SplitValue is "10MB" for [SplitBySize] and a number such
// as "5" for the count modes. The result is a ZIP archive.
func (c *Client) SplitBySizeOrCount(ctx context.Context, in FileInput, opts SplitSizeOptions) (*Result, error) {
	if opts.SplitValue == "" {
		return nil, fmt.Errorf("stirling: split by size or count: split value is required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "splitType", Value: int(opts.SplitType)},
		Param{Name: "splitValue", Value: opts.SplitValue},
	)
	return c.Call(ctx, "/api/v1/general/split-by-size-or-count", params...)
}

// ScalePages changes the page size and content scale of a PDF.
func (c *Client) ScalePages(ctx context.Context, in FileInput, opts ScaleOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "pageSize", Value: string(opts.PageSize)},
		Param{Name: "scaleFactor", Value: opts.ScaleFactor},
	)
	return c.Call(ctx, "/api/v1/general/scale-pages", params...)
}

// Rotate rotates every page by angle degrees. Stirling supports 0, 90, 180,
// and 270.
func (c *Client) Rotate(ctx context.Context, in FileInput, angle int) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "angle", Value: angle})
	return c.Call(ctx, "/api/v1/general/rotate-pdf", params...)
}

// RemovePages deletes the selected pages and returns the remaining PDF.
func (c *Client) RemovePages(ctx context.Context, in FileInput, pages string) (*Result, error) {
	if pages == "" {
		return nil, fmt.Errorf("stirling: remove pages: page numbers are required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "pageNumbers", Value: pages})
	return c.Call(ctx, "/api/v1/general/remove-pages", params...)
}

// RemoveImages strips images from a PDF and returns the smaller document.
func (c *Client) RemoveImages(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/general/remove-image-pdf", params...)
}

// RearrangePages changes the page order. Use [RearrangeCustom] with "3,1,2,4"
// or one of the built-in ordering modes.
func (c *Client) RearrangePages(ctx context.Context, in FileInput, opts RearrangeOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.Pages != "" {
		params = append(params, Param{Name: "pageNumbers", Value: opts.Pages})
	}
	if opts.Mode != "" {
		params = append(params, Param{Name: "customMode", Value: string(opts.Mode)})
	}
	return c.Call(ctx, "/api/v1/general/rearrange-pages", params...)
}

// ToSinglePage stacks all pages into one long page.
func (c *Client) ToSinglePage(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/general/pdf-to-single-page", params...)
}

// OverlayPDFs overlays one or more documents onto a base PDF. In
// [OverlaySequential] mode the overlay files are applied in order, in
// [OverlayInterleaved] mode they repeat, and in [OverlayFixedRepeat] mode
// Counts controls how often each overlay is repeated.
func (c *Client) OverlayPDFs(ctx context.Context, base FileInput, overlays []FileInput, opts OverlayOptions) (*Result, error) {
	params, err := inputParams(base)
	if err != nil {
		return nil, err
	}
	if len(overlays) == 0 {
		return nil, fmt.Errorf("stirling: overlay: at least one overlay file is required")
	}
	params = append(params,
		Param{Name: "overlayFiles", Value: overlays},
		Param{Name: "overlayMode", Value: string(opts.Mode)},
		Param{Name: "overlayPosition", Value: int(opts.Position)},
	)
	if len(opts.Counts) > 0 {
		params = append(params, Param{Name: "counts", Value: opts.Counts})
	}
	return c.Call(ctx, "/api/v1/general/overlay-pdfs", params...)
}

// MultiPageLayout puts several pages onto each sheet.
func (c *Client) MultiPageLayout(ctx context.Context, in FileInput, opts MultiPageOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "pagesPerSheet", Value: opts.PagesPerSheet},
		Param{Name: "addBorder", Value: opts.AddBorder},
	)
	return c.Call(ctx, "/api/v1/general/multi-page-layout", params...)
}

// Merge combines multiple PDFs into one document. The order depends on
// opts.SortType; with the default "orderProvided" it is the order of files.
func (c *Client) Merge(ctx context.Context, files []FileInput, opts *MergeOptions) (*Result, error) {
	params, err := multiInputParams("fileInput", files)
	if err != nil {
		return nil, err
	}
	if opts != nil {
		if opts.SortType != "" {
			params = append(params, Param{Name: "sortType", Value: string(opts.SortType)})
		}
		params = append(params,
			Param{Name: "removeCertSign", Value: opts.RemoveCertSign},
			Param{Name: "generateToc", Value: opts.GenerateTOC},
		)
	}
	return c.Call(ctx, "/api/v1/general/merge-pdfs", params...)
}

// ExtractBookmarks returns the document outline as a JSON object.
func (c *Client) ExtractBookmarks(ctx context.Context, in FileInput) (map[string]any, error) {
	if in == nil {
		return nil, fmt.Errorf("stirling: extract bookmarks: an input file is required")
	}
	params := []Param{{Name: "file", Value: in}}
	if id, ok := FileID(in); ok {
		params = []Param{{Name: "fileId", Value: id}}
	}
	return c.CallJSON(ctx, "/api/v1/general/extract-bookmarks", params...)
}

// EditTableOfContents replaces a PDF's bookmark tree. BookmarkData is the JSON
// document expected by Stirling PDF.
func (c *Client) EditTableOfContents(ctx context.Context, in FileInput, bookmarkData string, replaceExisting bool) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "bookmarkData", Value: bookmarkData},
		Param{Name: "replaceExisting", Value: replaceExisting},
	)
	return c.Call(ctx, "/api/v1/general/edit-table-of-contents", params...)
}

// Crop trims every page to the rectangle described by opts.
func (c *Client) Crop(ctx context.Context, in FileInput, opts CropOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "x", Value: opts.X},
		Param{Name: "y", Value: opts.Y},
		Param{Name: "width", Value: opts.Width},
		Param{Name: "height", Value: opts.Height},
	)
	return c.Call(ctx, "/api/v1/general/crop", params...)
}

// Booklet imposes pages for booklet printing.
func (c *Client) Booklet(ctx context.Context, in FileInput, opts BookletOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "bookletType", Value: string(opts.Type)},
		Param{Name: "pagesPerSheet", Value: opts.PagesPerSheet},
		Param{Name: "addBorder", Value: opts.AddBorder},
		Param{Name: "pageOrientation", Value: string(opts.Orientation)},
	)
	return c.Call(ctx, "/api/v1/general/booklet-imposition", params...)
}
