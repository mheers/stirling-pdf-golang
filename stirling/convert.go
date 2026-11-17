package stirling

import (
	"context"
	"fmt"
)

// PDFToImagesOptions configures [Client.PDFToImages]. Pages uses Stirling's
// page syntax and defaults to "all". DPI defaults to 300.
type PDFToImagesOptions struct {
	Pages     string
	Format    ImageFormat
	Output    ImageOutput
	ColorType ColorType
	DPI       *int
}

// ImagesToPDFOptions configures [Client.ImagesToPDF].
type ImagesToPDFOptions struct {
	Fit        ImageFit
	ColorType  ColorType
	AutoRotate *bool
}

// HTMLToPDFOptions configures [Client.HTMLToPDF]. Zoom is a percentage-style
// scale factor, where 1 is 100%.
type HTMLToPDFOptions struct {
	Zoom *float64
}

// EMLToPDFOptions configures [Client.EMLToPDF].
type EMLToPDFOptions struct {
	IncludeAttachments   *bool
	MaxAttachmentSizeMB  *int
	DownloadHTML         *bool
	IncludeAllRecipients *bool
}

// ConvertURL fetches a web page and renders it as a PDF.
func (c *Client) ConvertURL(ctx context.Context, url string) (*Result, error) {
	if url == "" {
		return nil, fmt.Errorf("stirling: convert URL: URL is required")
	}
	return c.Call(ctx, "/api/v1/convert/url/pdf", Param{Name: "urlInput", Value: url})
}

// PDFToXML converts a PDF to XML.
func (c *Client) PDFToXML(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/convert/pdf/xml", params...)
}

// PDFToWord converts a PDF to a word-processing document.
func (c *Client) PDFToWord(ctx context.Context, in FileInput, format PDFToWordFormat) (*Result, error) {
	if format == "" {
		return nil, fmt.Errorf("stirling: PDF to word: output format is required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "outputFormat", Value: string(format)})
	return c.Call(ctx, "/api/v1/convert/pdf/word", params...)
}

// PDFToText converts a PDF to plain text or RTF.
func (c *Client) PDFToText(ctx context.Context, in FileInput, format PDFToTextFormat) (*Result, error) {
	if format == "" {
		return nil, fmt.Errorf("stirling: PDF to text: output format is required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "outputFormat", Value: string(format)})
	return c.Call(ctx, "/api/v1/convert/pdf/text", params...)
}

// PDFToPresentation converts a PDF to a presentation document.
func (c *Client) PDFToPresentation(ctx context.Context, in FileInput, format PDFToPresentationFormat) (*Result, error) {
	if format == "" {
		return nil, fmt.Errorf("stirling: PDF to presentation: output format is required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "outputFormat", Value: string(format)})
	return c.Call(ctx, "/api/v1/convert/pdf/presentation", params...)
}

// PDFToPDFA converts a PDF to a PDF/A profile.
func (c *Client) PDFToPDFA(ctx context.Context, in FileInput, format PDFToPDFAFormat) (*Result, error) {
	if format == "" {
		return nil, fmt.Errorf("stirling: PDF to PDF/A: output format is required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "outputFormat", Value: string(format)})
	return c.Call(ctx, "/api/v1/convert/pdf/pdfa", params...)
}

// PDFToMarkdown converts a PDF to Markdown.
func (c *Client) PDFToMarkdown(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/convert/pdf/markdown", params...)
}

// PDFToImages renders PDF pages as PNG, JPEG, GIF, or WebP images. With
// [ImageSingle] the server returns one image; with [ImageMultiple] it returns a
// ZIP archive.
func (c *Client) PDFToImages(ctx context.Context, in FileInput, opts PDFToImagesOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.Pages != "" {
		params = append(params, Param{Name: "pageNumbers", Value: opts.Pages})
	}
	if opts.Format != "" {
		params = append(params, Param{Name: "imageFormat", Value: string(opts.Format)})
	}
	if opts.Output != "" {
		params = append(params, Param{Name: "singleOrMultiple", Value: string(opts.Output)})
	}
	if opts.ColorType != "" {
		params = append(params, Param{Name: "colorType", Value: string(opts.ColorType)})
	}
	params = append(params, Param{Name: "dpi", Value: opts.DPI})
	return c.Call(ctx, "/api/v1/convert/pdf/img", params...)
}

// PDFToHTML converts a PDF to HTML.
func (c *Client) PDFToHTML(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/convert/pdf/html", params...)
}

// PDFToCSV extracts tables from a PDF as CSV.
func (c *Client) PDFToCSV(ctx context.Context, in FileInput, pages string) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if pages != "" {
		params = append(params, Param{Name: "pageNumbers", Value: pages})
	}
	return c.Call(ctx, "/api/v1/convert/pdf/csv", params...)
}

// MarkdownToPDF renders a Markdown document as a PDF.
func (c *Client) MarkdownToPDF(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/convert/markdown/pdf", params...)
}

// ImagesToPDF combines images into a PDF.
func (c *Client) ImagesToPDF(ctx context.Context, files []FileInput, opts ImagesToPDFOptions) (*Result, error) {
	params, err := multiInputParams("fileInput", files)
	if err != nil {
		return nil, err
	}
	if opts.Fit != "" {
		params = append(params, Param{Name: "fitOption", Value: string(opts.Fit)})
	}
	if opts.ColorType != "" {
		params = append(params, Param{Name: "colorType", Value: string(opts.ColorType)})
	}
	params = append(params, Param{Name: "autoRotate", Value: opts.AutoRotate})
	return c.Call(ctx, "/api/v1/convert/img/pdf", params...)
}

// HTMLToPDF converts an HTML file to a PDF.
func (c *Client) HTMLToPDF(ctx context.Context, in FileInput, opts *HTMLToPDFOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts != nil {
		params = append(params, Param{Name: "zoom", Value: opts.Zoom})
	}
	return c.Call(ctx, "/api/v1/convert/html/pdf", params...)
}

// FileToPDF converts a supported office or text file to a PDF.
func (c *Client) FileToPDF(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/convert/file/pdf", params...)
}

// EMLToPDF converts an email file to a PDF.
func (c *Client) EMLToPDF(ctx context.Context, in FileInput, opts *EMLToPDFOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts != nil {
		params = append(params,
			Param{Name: "includeAttachments", Value: opts.IncludeAttachments},
			Param{Name: "maxAttachmentSizeMB", Value: opts.MaxAttachmentSizeMB},
			Param{Name: "downloadHtml", Value: opts.DownloadHTML},
			Param{Name: "includeAllRecipients", Value: opts.IncludeAllRecipients},
		)
	}
	return c.Call(ctx, "/api/v1/convert/eml/pdf", params...)
}
