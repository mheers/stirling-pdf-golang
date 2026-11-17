package stirling

import (
	"context"
	"fmt"
)

// SignatureValidationOptions configures [Client.ValidateSignature].
type SignatureValidationOptions struct {
	CertFile FileInput
}

// SanitizeOptions configures [Client.Sanitize]. Nil flags keep the server
// defaults.
type SanitizeOptions struct {
	RemoveJavaScript  *bool
	RemoveEmbedded    *bool
	RemoveXMPMetadata *bool
	RemoveMetadata    *bool
	RemoveLinks       *bool
	RemoveFonts       *bool
}

// RedactionArea is a rectangular redaction on a page. Coordinates are in PDF
// points; Page is one-based.
type RedactionArea struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
	Page   int
	Color  string
}

// RedactManualOptions configures [Client.RedactManual].
type RedactManualOptions struct {
	Pages          string
	Redactions     []RedactionArea
	ConvertToImage bool
	RedactionColor string
}

// RedactAutoOptions configures [Client.RedactAuto]. Texts is a
// comma-separated list passed as listOfText; use [Client.RedactManual] for
// coordinate-based redaction.
type RedactAutoOptions struct {
	UseRegex       *bool
	WholeWord      *bool
	Color          string
	Padding        *float64
	ConvertToImage *bool
}

// SignPDFOptions configures [Client.SignPDF]. CertType selects which
// certificate fields are used: [CertPEM] needs PrivateKeyFile and CertFile,
// [CertPKCS12] needs P12File, [CertJKS] needs JKSFile, and [CertServer] uses
// the server's own certificate.
type SignPDFOptions struct {
	CertType       CertType
	PrivateKeyFile FileInput
	CertFile       FileInput
	P12File        FileInput
	JKSFile        FileInput
	Password       string
	ShowSignature  *bool
	Reason         string
	Location       string
	Name           string
	PageNumber     *int
	ShowLogo       *bool
}

// WatermarkOptions configures [Client.AddWatermark]. Set Type to
// [WatermarkText] and Text, or [WatermarkImage] and Image.
type WatermarkOptions struct {
	Type           WatermarkType
	Text           string
	Image          FileInput
	Alphabet       Alphabet
	FontSize       *float64
	Rotation       *float64
	Opacity        *float64
	WidthSpacer    *int
	HeightSpacer   *int
	Color          string
	ConvertToImage *bool
}

// PasswordOptions configures [Client.AddPassword].
type PasswordOptions struct {
	UserPassword                   string
	OwnerPassword                  string
	KeyLength                      *int
	PreventAssembly                *bool
	PreventExtractContent          *bool
	PreventExtractForAccessibility *bool
	PreventFillInForm              *bool
	PreventModify                  *bool
	PreventModifyAnnotations       *bool
	PreventPrinting                *bool
	PreventPrintingFaithful        *bool
}

// ValidateSignature validates a PDF signature, optionally against a
// certificate file, and returns the server's JSON report.
func (c *Client) ValidateSignature(ctx context.Context, in FileInput, opts *SignatureValidationOptions) (map[string]any, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts != nil && opts.CertFile != nil {
		params = append(params, Param{Name: "certFile", Value: opts.CertFile})
	}
	return c.CallJSON(ctx, "/api/v1/security/validate-signature", params...)
}

// Sanitize removes scripts, metadata, links, fonts, or embedded files from a
// PDF.
func (c *Client) Sanitize(ctx context.Context, in FileInput, opts *SanitizeOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts != nil {
		params = append(params,
			Param{Name: "removeJavaScript", Value: opts.RemoveJavaScript},
			Param{Name: "removeEmbeddedFiles", Value: opts.RemoveEmbedded},
			Param{Name: "removeXMPMetadata", Value: opts.RemoveXMPMetadata},
			Param{Name: "removeMetadata", Value: opts.RemoveMetadata},
			Param{Name: "removeLinks", Value: opts.RemoveLinks},
			Param{Name: "removeFonts", Value: opts.RemoveFonts},
		)
	}
	return c.Call(ctx, "/api/v1/security/sanitize-pdf", params...)
}

// RemovePassword removes password protection from a PDF.
func (c *Client) RemovePassword(ctx context.Context, in FileInput, password string) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if password != "" {
		params = append(params, Param{Name: "password", Value: password})
	}
	return c.Call(ctx, "/api/v1/security/remove-password", params...)
}

// RemoveCertSign removes certification signatures from a PDF.
func (c *Client) RemoveCertSign(ctx context.Context, in FileInput) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "/api/v1/security/remove-cert-sign", params...)
}

// RedactManual applies coordinate-based redactions to a PDF.
func (c *Client) RedactManual(ctx context.Context, in FileInput, opts RedactManualOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.Pages != "" {
		params = append(params, Param{Name: "pageNumbers", Value: opts.Pages})
	}
	if opts.RedactionColor != "" {
		params = append(params, Param{Name: "pageRedactionColor", Value: opts.RedactionColor})
	}
	if len(opts.Redactions) > 0 {
		objects := make([]map[string]any, 0, len(opts.Redactions))
		for _, area := range opts.Redactions {
			objects = append(objects, map[string]any{
				"x":      area.X,
				"y":      area.Y,
				"width":  area.Width,
				"height": area.Height,
				"page":   area.Page,
				"color":  area.Color,
			})
		}
		params = append(params, Param{Name: "redactions", Value: objects})
	}
	params = append(params, Param{Name: "convertPDFToImage", Value: opts.ConvertToImage})
	return c.Call(ctx, "/api/v1/security/redact", params...)
}

// PDFInfo returns security information about a PDF as a JSON object.
func (c *Client) PDFInfo(ctx context.Context, in FileInput) (map[string]any, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	return c.CallJSON(ctx, "/api/v1/security/get-info-on-pdf", params...)
}

// SignPDF adds a digital signature to a PDF.
func (c *Client) SignPDF(ctx context.Context, in FileInput, opts SignPDFOptions) (*Result, error) {
	if opts.CertType == "" {
		return nil, fmt.Errorf("stirling: sign PDF: certificate type is required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "certType", Value: string(opts.CertType)})
	for _, item := range []struct {
		name string
		file FileInput
	}{
		{"privateKeyFile", opts.PrivateKeyFile},
		{"certFile", opts.CertFile},
		{"p12File", opts.P12File},
		{"jksFile", opts.JKSFile},
	} {
		if item.file != nil {
			params = append(params, Param{Name: item.name, Value: item.file})
		}
	}
	if opts.Password != "" {
		params = append(params, Param{Name: "password", Value: opts.Password})
	}
	if opts.Reason != "" {
		params = append(params, Param{Name: "reason", Value: opts.Reason})
	}
	if opts.Location != "" {
		params = append(params, Param{Name: "location", Value: opts.Location})
	}
	if opts.Name != "" {
		params = append(params, Param{Name: "name", Value: opts.Name})
	}
	params = append(params,
		Param{Name: "showSignature", Value: opts.ShowSignature},
		Param{Name: "pageNumber", Value: opts.PageNumber},
		Param{Name: "showLogo", Value: opts.ShowLogo},
	)
	return c.Call(ctx, "/api/v1/security/cert-sign", params...)
}

// RedactAuto redacts all occurrences of texts from a PDF. Texts is a
// comma-separated list.
func (c *Client) RedactAuto(ctx context.Context, in FileInput, texts string, opts RedactAutoOptions) (*Result, error) {
	if texts == "" {
		return nil, fmt.Errorf("stirling: redact: text to redact is required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "listOfText", Value: texts})
	if opts.Color != "" {
		params = append(params, Param{Name: "redactColor", Value: opts.Color})
	}
	params = append(params,
		Param{Name: "useRegex", Value: opts.UseRegex},
		Param{Name: "wholeWordSearch", Value: opts.WholeWord},
		Param{Name: "customPadding", Value: opts.Padding},
		Param{Name: "convertPDFToImage", Value: opts.ConvertToImage},
	)
	return c.Call(ctx, "/api/v1/security/auto-redact", params...)
}

// AddWatermark adds a text or image watermark to a PDF.
func (c *Client) AddWatermark(ctx context.Context, in FileInput, opts WatermarkOptions) (*Result, error) {
	if opts.Type == "" {
		return nil, fmt.Errorf("stirling: add watermark: watermark type is required")
	}
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "watermarkType", Value: string(opts.Type)})
	if opts.Text != "" {
		params = append(params, Param{Name: "watermarkText", Value: opts.Text})
	}
	if opts.Image != nil {
		params = append(params, Param{Name: "watermarkImage", Value: opts.Image})
	}
	if opts.Alphabet != "" {
		params = append(params, Param{Name: "alphabet", Value: string(opts.Alphabet)})
	}
	if opts.Color != "" {
		params = append(params, Param{Name: "customColor", Value: opts.Color})
	}
	params = append(params,
		Param{Name: "fontSize", Value: opts.FontSize},
		Param{Name: "rotation", Value: opts.Rotation},
		Param{Name: "opacity", Value: opts.Opacity},
		Param{Name: "widthSpacer", Value: opts.WidthSpacer},
		Param{Name: "heightSpacer", Value: opts.HeightSpacer},
		Param{Name: "convertPDFToImage", Value: opts.ConvertToImage},
	)
	return c.Call(ctx, "/api/v1/security/add-watermark", params...)
}

// AddPassword encrypts a PDF with user and/or owner passwords. KeyLength is
// 40, 128, or 256; the server default is 256.
func (c *Client) AddPassword(ctx context.Context, in FileInput, opts PasswordOptions) (*Result, error) {
	params, err := inputParams(in)
	if err != nil {
		return nil, err
	}
	if opts.UserPassword != "" {
		params = append(params, Param{Name: "password", Value: opts.UserPassword})
	}
	if opts.OwnerPassword != "" {
		params = append(params, Param{Name: "ownerPassword", Value: opts.OwnerPassword})
	}
	params = append(params,
		Param{Name: "keyLength", Value: opts.KeyLength},
		Param{Name: "preventAssembly", Value: opts.PreventAssembly},
		Param{Name: "preventExtractContent", Value: opts.PreventExtractContent},
		Param{Name: "preventExtractForAccessibility", Value: opts.PreventExtractForAccessibility},
		Param{Name: "preventFillInForm", Value: opts.PreventFillInForm},
		Param{Name: "preventModify", Value: opts.PreventModify},
		Param{Name: "preventModifyAnnotations", Value: opts.PreventModifyAnnotations},
		Param{Name: "preventPrinting", Value: opts.PreventPrinting},
		Param{Name: "preventPrintingFaithful", Value: opts.PreventPrintingFaithful},
	)
	return c.Call(ctx, "/api/v1/security/add-password", params...)
}
