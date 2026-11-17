package stirling

import (
	"context"
	"fmt"
)

// MetadataOptions configures [Client.UpdateMetadata]. Empty string fields are
// not sent, so only the metadata you set is changed. Custom is mapped to the
// server's free-form allRequestParams field; custom keys should start with
// customKey or customValue.
type MetadataOptions struct {
	DeleteAll        bool
	Author           string
	CreationDate     string
	Creator          string
	Keywords         string
	ModificationDate string
	Producer         string
	Subject          string
	Title            string
	Trapped          string
	Custom           map[string]string
}

// ScannerEffectOptions configures [Client.ScannerEffect]. Nil fields keep the
// server defaults.
type ScannerEffectOptions struct {
	Quality         ScanQuality
	Rotation        ScanRotation
	Colorspace      ColorType
	Border          *int
	Rotate          *int
	RotateVariance  *int
	Brightness      *float64
	Contrast        *float64
	Blur            *float64
	Noise           *float64
	Yellowish       *bool
	Resolution      *int
	QualityValue    *int
	RotationValue   *int
	AdvancedEnabled *bool
}

// ReplaceInvertColorsOptions configures [Client.ReplaceInvertColors].
type ReplaceInvertColorsOptions struct {
	Option          ReplaceColorOption
	Combination     HighContrastCombination
	BackgroundColor string
	TextColor       string
}

// RemoveBlanksOptions configures [Client.RemoveBlankPages].
type RemoveBlanksOptions struct {
	Threshold    int
	WhitePercent float64
}

// OCROptions configures [Client.OCR]. Languages defaults to ["eng"] on the
// server when nil.
type OCROptions struct {
	Languages         []string
	Sidecar           *bool
	Deskew            *bool
	Clean             *bool
	CleanFinal        *bool
	Type              OCRType
	RenderType        OCRRenderType
	RemoveImagesAfter *bool
}

// ExtractImagesOptions configures [Client.ExtractImages].
type ExtractImagesOptions struct {
	Format          ImageFormat
	AllowDuplicates *bool
}

// ExtractImageScansOptions configures [Client.ExtractImageScans].
type ExtractImageScansOptions struct {
	AngleThreshold int
	Tolerance      int
	MinArea        int
	MinContourArea int
	BorderSize     int
}

// CompressOptions configures [Client.Compress]. OptimizeLevel ranges from 1
// (least) to 9 (most compression). ExpectedOutputSize accepts values such as
// "25KB", "10MB", or "1GB".
type CompressOptions struct {
	OptimizeLevel      *int
	ExpectedOutputSize string
	Linearize          *bool
	Normalize          *bool
	Grayscale          *bool
}

// AutoSplitOptions configures [Client.AutoSplit].
type AutoSplitOptions struct {
	DuplexMode *bool
}

// AutoRenameOptions configures [Client.AutoRename].
type AutoRenameOptions struct {
	UseFirstTextAsFallback *bool
}

// StampOptions configures [Client.AddStamp]. Set Type to [StampImage] and
// provide Image for an image stamp, or [StampText] and Text for a text stamp.
type StampOptions struct {
	Pages        string
	Type         StampType
	Text         string
	Image        FileInput
	Alphabet     Alphabet
	FontSize     *float64
	Rotation     *float64
	Opacity      *float64
	Position     *int
	OverrideX    *float64
	OverrideY    *float64
	CustomMargin Margin
	CustomColor  string
}

// PageNumbersOptions configures [Client.AddPageNumbers].
type PageNumbersOptions struct {
	Pages          string
	CustomMargin   Margin
	FontSize       *float64
	FontType       FontType
	Position       *int
	StartingNumber *int
	PagesToNumber  string
	CustomText     string
}

// AddImageOptions configures [Client.AddImage].
type AddImageOptions struct {
	Image     FileInput
	X         float64
	Y         float64
	EveryPage bool
}

// UpdateMetadata changes document metadata and returns the PDF.
func (c *Client) UpdateMetadata(ctx context.Context, in FileInput, opts MetadataOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "deleteAll", Value: opts.DeleteAll})
	for _, item := range []struct {
		name  string
		value string
	}{
		{"author", opts.Author},
		{"creationDate", opts.CreationDate},
		{"creator", opts.Creator},
		{"keywords", opts.Keywords},
		{"modificationDate", opts.ModificationDate},
		{"producer", opts.Producer},
		{"subject", opts.Subject},
		{"title", opts.Title},
		{"trapped", opts.Trapped},
	} {
		if item.value != "" {
			params = append(params, Param{Name: item.name, Value: item.value})
		}
	}
	if len(opts.Custom) > 0 {
		params = append(params, Param{Name: "allRequestParams", Value: opts.Custom})
	}
	return c.Call(ctx, "/api/v1/misc/update-metadata", params...)
}

// UnlockForms removes form-field protections from a PDF.
func (c *Client) UnlockForms(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/misc/unlock-pdf-forms", params...)
}

// ShowJavaScript extracts embedded JavaScript from a PDF as plain text.
func (c *Client) ShowJavaScript(ctx context.Context, in FileInput) (string, error) {
	params, err := inputParams(in)
	if err != nil {
		return "", err
	}
	return c.CallText(ctx, "/api/v1/misc/show-javascript", params...)
}

// ScannerEffect applies a scanned-paper look to a PDF.
func (c *Client) ScannerEffect(ctx context.Context, in FileInput, opts ScannerEffectOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.Quality != "" {
		params = append(params, Param{Name: "quality", Value: string(opts.Quality)})
	}
	if opts.Rotation != "" {
		params = append(params, Param{Name: "rotation", Value: string(opts.Rotation)})
	}
	if opts.Colorspace != "" {
		params = append(params, Param{Name: "colorspace", Value: string(opts.Colorspace)})
	}
	params = append(params,
		Param{Name: "border", Value: opts.Border},
		Param{Name: "rotate", Value: opts.Rotate},
		Param{Name: "rotateVariance", Value: opts.RotateVariance},
		Param{Name: "brightness", Value: opts.Brightness},
		Param{Name: "contrast", Value: opts.Contrast},
		Param{Name: "blur", Value: opts.Blur},
		Param{Name: "noise", Value: opts.Noise},
		Param{Name: "yellowish", Value: opts.Yellowish},
		Param{Name: "resolution", Value: opts.Resolution},
		Param{Name: "advancedEnabled", Value: opts.AdvancedEnabled},
		Param{Name: "qualityValue", Value: opts.QualityValue},
		Param{Name: "rotationValue", Value: opts.RotationValue},
	)
	return c.Call(ctx, "/api/v1/misc/scanner-effect", params...)
}

// ReplaceInvertColors replaces colors or inverts a PDF for accessibility.
func (c *Client) ReplaceInvertColors(ctx context.Context, in FileInput, opts ReplaceInvertColorsOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.Option != "" {
		params = append(params, Param{Name: "replaceAndInvertOption", Value: string(opts.Option)})
	}
	if opts.Combination != "" {
		params = append(params, Param{Name: "highContrastColorCombination", Value: string(opts.Combination)})
	}
	if opts.BackgroundColor != "" {
		params = append(params, Param{Name: "backGroundColor", Value: opts.BackgroundColor})
	}
	if opts.TextColor != "" {
		params = append(params, Param{Name: "textColor", Value: opts.TextColor})
	}
	return c.Call(ctx, "/api/v1/misc/replace-invert-pdf", params...)
}

// Repair attempts to repair a damaged PDF.
func (c *Client) Repair(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/misc/repair", params...)
}

// RemoveBlankPages removes pages that are mostly blank.
func (c *Client) RemoveBlankPages(ctx context.Context, in FileInput, opts RemoveBlanksOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "threshold", Value: opts.Threshold},
		Param{Name: "whitePercent", Value: opts.WhitePercent},
	)
	return c.Call(ctx, "/api/v1/misc/remove-blanks", params...)
}

// OCR runs optical character recognition on a PDF. Use Languages such as
// ["eng"] or ["eng", "deu"].
func (c *Client) OCR(ctx context.Context, in FileInput, opts OCROptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if len(opts.Languages) > 0 {
		params = append(params, Param{Name: "languages", Value: opts.Languages})
	}
	if opts.Type != "" {
		params = append(params, Param{Name: "ocrType", Value: string(opts.Type)})
	}
	if opts.RenderType != "" {
		params = append(params, Param{Name: "ocrRenderType", Value: string(opts.RenderType)})
	}
	params = append(params,
		Param{Name: "sidecar", Value: opts.Sidecar},
		Param{Name: "deskew", Value: opts.Deskew},
		Param{Name: "clean", Value: opts.Clean},
		Param{Name: "cleanFinal", Value: opts.CleanFinal},
		Param{Name: "removeImagesAfter", Value: opts.RemoveImagesAfter},
	)
	return c.Call(ctx, "/api/v1/misc/ocr-pdf", params...)
}

// Flatten merges form fields and annotations into the page content. Set
// onlyForms to true to flatten forms but keep other annotations.
func (c *Client) Flatten(ctx context.Context, in FileInput, onlyForms bool) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "flattenOnlyForms", Value: onlyForms})
	return c.Call(ctx, "/api/v1/misc/flatten", params...)
}

// ExtractImages extracts embedded images as a ZIP archive or a single image.
func (c *Client) ExtractImages(ctx context.Context, in FileInput, opts ExtractImagesOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.Format != "" {
		params = append(params, Param{Name: "format", Value: string(opts.Format)})
	}
	params = append(params, Param{Name: "allowDuplicates", Value: opts.AllowDuplicates})
	return c.Call(ctx, "/api/v1/misc/extract-images", params...)
}

// ExtractImageScans detects scanned images inside a PDF and extracts them.
func (c *Client) ExtractImageScans(ctx context.Context, in FileInput, opts ExtractImageScansOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "angleThreshold", Value: opts.AngleThreshold},
		Param{Name: "tolerance", Value: opts.Tolerance},
		Param{Name: "minArea", Value: opts.MinArea},
		Param{Name: "minContourArea", Value: opts.MinContourArea},
		Param{Name: "borderSize", Value: opts.BorderSize},
	)
	return c.Call(ctx, "/api/v1/misc/extract-image-scans", params...)
}

// Decompress removes compression from a PDF's streams.
func (c *Client) Decompress(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/misc/decompress-pdf", params...)
}

// Compress optimizes a PDF. Pass nil options for the server defaults.
func (c *Client) Compress(ctx context.Context, in FileInput, opts *CompressOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts != nil {
		if opts.ExpectedOutputSize != "" {
			params = append(params, Param{Name: "expectedOutputSize", Value: opts.ExpectedOutputSize})
		}
		params = append(params,
			Param{Name: "optimizeLevel", Value: opts.OptimizeLevel},
			Param{Name: "linearize", Value: opts.Linearize},
			Param{Name: "normalize", Value: opts.Normalize},
			Param{Name: "grayscale", Value: opts.Grayscale},
		)
	}
	return c.Call(ctx, "/api/v1/misc/compress-pdf", params...)
}

// AutoSplit splits scanned batches at blank pages.
func (c *Client) AutoSplit(ctx context.Context, in FileInput, opts *AutoSplitOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts != nil {
		params = append(params, Param{Name: "duplexMode", Value: opts.DuplexMode})
	}
	return c.Call(ctx, "/api/v1/misc/auto-split-pdf", params...)
}

// AutoRename derives a filename from a PDF's header text. The server returns
// text, so the result is exposed as a [Result].
func (c *Client) AutoRename(ctx context.Context, in FileInput, opts *AutoRenameOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts != nil {
		params = append(params, Param{Name: "useFirstTextAsFallback", Value: opts.UseFirstTextAsFallback})
	}
	return c.Call(ctx, "/api/v1/misc/auto-rename", params...)
}

// AddStamp draws a text or image stamp on selected pages.
func (c *Client) AddStamp(ctx context.Context, in FileInput, opts StampOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.Pages != "" {
		params = append(params, Param{Name: "pageNumbers", Value: opts.Pages})
	}
	if opts.Type != "" {
		params = append(params, Param{Name: "stampType", Value: string(opts.Type)})
	}
	if opts.Text != "" {
		params = append(params, Param{Name: "stampText", Value: opts.Text})
	}
	if opts.Image != nil {
		params = append(params, Param{Name: "stampImage", Value: opts.Image})
	}
	if opts.Alphabet != "" {
		params = append(params, Param{Name: "alphabet", Value: string(opts.Alphabet)})
	}
	if opts.CustomMargin != "" {
		params = append(params, Param{Name: "customMargin", Value: string(opts.CustomMargin)})
	}
	if opts.CustomColor != "" {
		params = append(params, Param{Name: "customColor", Value: opts.CustomColor})
	}
	params = append(params,
		Param{Name: "fontSize", Value: opts.FontSize},
		Param{Name: "rotation", Value: opts.Rotation},
		Param{Name: "opacity", Value: opts.Opacity},
		Param{Name: "position", Value: opts.Position},
		Param{Name: "overrideX", Value: opts.OverrideX},
		Param{Name: "overrideY", Value: opts.OverrideY},
	)
	return c.Call(ctx, "/api/v1/misc/add-stamp", params...)
}

// AddPageNumbers writes page numbers on a PDF.
func (c *Client) AddPageNumbers(ctx context.Context, in FileInput, opts PageNumbersOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.Pages != "" {
		params = append(params, Param{Name: "pageNumbers", Value: opts.Pages})
	}
	if opts.CustomMargin != "" {
		params = append(params, Param{Name: "customMargin", Value: string(opts.CustomMargin)})
	}
	if opts.FontType != "" {
		params = append(params, Param{Name: "fontType", Value: string(opts.FontType)})
	}
	if opts.PagesToNumber != "" {
		params = append(params, Param{Name: "pagesToNumber", Value: opts.PagesToNumber})
	}
	if opts.CustomText != "" {
		params = append(params, Param{Name: "customText", Value: opts.CustomText})
	}
	params = append(params,
		Param{Name: "fontSize", Value: opts.FontSize},
		Param{Name: "position", Value: opts.Position},
		Param{Name: "startingNumber", Value: opts.StartingNumber},
	)
	return c.Call(ctx, "/api/v1/misc/add-page-numbers", params...)
}

// AddImage places an image on a PDF page. Coordinates are in PDF points.
func (c *Client) AddImage(ctx context.Context, in FileInput, opts AddImageOptions) (*Result, error) {
	if opts.Image == nil {
		return nil, fmt.Errorf("stirling: add image: an image file is required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params,
		Param{Name: "imageFile", Value: opts.Image},
		Param{Name: "x", Value: opts.X},
		Param{Name: "y", Value: opts.Y},
		Param{Name: "everyPage", Value: opts.EveryPage},
	)
	return c.Call(ctx, "/api/v1/misc/add-image", params...)
}

// AddAttachments embeds one or more files into a PDF.
func (c *Client) AddAttachments(ctx context.Context, in FileInput, attachments []FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if len(attachments) == 0 {
		return nil, fmt.Errorf("stirling: add attachments: at least one attachment is required")
	}
	params = append(params, Param{Name: "attachments", Value: attachments})
	return c.Call(ctx, "/api/v1/misc/add-attachments", params...)
}
