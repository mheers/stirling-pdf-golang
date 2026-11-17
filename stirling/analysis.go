package stirling

import "context"

// SecurityInfo returns encryption and permission information about a PDF.
func (c *Client) SecurityInfo(ctx context.Context, in FileInput) (map[string]any, error) {
	return c.analysis(ctx, "/api/v1/analysis/security-info", in)
}

// PageDimensions returns the dimensions of every page as JSON.
func (c *Client) PageDimensions(ctx context.Context, in FileInput) (map[string]any, error) {
	return c.analysis(ctx, "/api/v1/analysis/page-dimensions", in)
}

// PageCount returns the number of pages as JSON.
func (c *Client) PageCount(ctx context.Context, in FileInput) (map[string]any, error) {
	return c.analysis(ctx, "/api/v1/analysis/page-count", in)
}

// FormFields returns the PDF's form fields as JSON.
func (c *Client) FormFields(ctx context.Context, in FileInput) (map[string]any, error) {
	return c.analysis(ctx, "/api/v1/analysis/form-fields", in)
}

// FontInfo returns the fonts used by a PDF as JSON.
func (c *Client) FontInfo(ctx context.Context, in FileInput) (map[string]any, error) {
	return c.analysis(ctx, "/api/v1/analysis/font-info", in)
}

// DocumentProperties returns document metadata as JSON.
func (c *Client) DocumentProperties(ctx context.Context, in FileInput) (map[string]any, error) {
	return c.analysis(ctx, "/api/v1/analysis/document-properties", in)
}

// BasicInfo returns basic PDF information such as version and producer.
func (c *Client) BasicInfo(ctx context.Context, in FileInput) (map[string]any, error) {
	return c.analysis(ctx, "/api/v1/analysis/basic-info", in)
}

// AnnotationInfo returns the PDF's annotations as JSON.
func (c *Client) AnnotationInfo(ctx context.Context, in FileInput) (map[string]any, error) {
	return c.analysis(ctx, "/api/v1/analysis/annotation-info", in)
}

func (c *Client) analysis(ctx context.Context, path string, in FileInput) (map[string]any, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.CallJSON(ctx, path, params...)
}
