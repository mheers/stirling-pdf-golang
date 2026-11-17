package stirling

import (
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type capturedRequest struct {
	Method string
	Path   string
	Values url.Values
	Files  map[string][]*multipart.FileHeader
}

type captureServer struct {
	mu       sync.Mutex
	requests []capturedRequest
	server   *httptest.Server
}

func newCaptureServer(t *testing.T) *captureServer {
	t.Helper()
	capture := &captureServer{}
	capture.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := capturedRequest{Method: r.Method, Path: r.URL.Path, Files: map[string][]*multipart.FileHeader{}}
		if r.Method == http.MethodPost {
			if err := r.ParseMultipartForm(8 << 20); err != nil {
				t.Errorf("ParseMultipartForm: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			request.Values = r.MultipartForm.Value
			request.Files = r.MultipartForm.File
		}
		capture.mu.Lock()
		capture.requests = append(capture.requests, request)
		capture.mu.Unlock()

		switch {
		case r.URL.Path == "/api/v1/info/status":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"3.3.0","status":"UP"}`))
		case strings.Contains(r.URL.Path, "/analysis/"),
			r.URL.Path == "/api/v1/security/validate-signature",
			r.URL.Path == "/api/v1/security/get-info-on-pdf",
			r.URL.Path == "/api/v1/general/extract-bookmarks":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.URL.Path == "/api/v1/misc/show-javascript", r.URL.Path == "/api/v1/misc/auto-rename":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("text result"))
		default:
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.7 result"))
		}
	}))
	t.Cleanup(capture.server.Close)
	return capture
}

func (c *captureServer) last(t *testing.T) capturedRequest {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.requests) == 0 {
		t.Fatal("no request captured")
	}
	return c.requests[len(c.requests)-1]
}

func testPDF(name string) FileInput { return FromBytes(name, []byte("%PDF-1.7")) }

func TestOperationPathsAndFields(t *testing.T) {
	type testCase struct {
		name  string
		path  string
		call  func(context.Context, *Client) error
		want  map[string]string
		files map[string][]string
	}
	cases := []testCase{
		{
			name: "SplitBySections", path: "/api/v1/general/split-pdf-by-sections",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.SplitBySections(ctx, testPDF("a.pdf"), SplitSectionsOptions{HorizontalDivisions: 2, VerticalDivisions: 3, Merge: true})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"horizontalDivisions": "2", "verticalDivisions": "3", "merge": "true"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "SplitByChapters", path: "/api/v1/general/split-pdf-by-chapters",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.SplitByChapters(ctx, testPDF("a.pdf"), SplitChaptersOptions{BookmarkLevel: Ptr(2)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"bookmarkLevel": "2"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "SplitPages", path: "/api/v1/general/split-pages",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.SplitPages(ctx, testPDF("a.pdf"), "1-3")
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"pageNumbers": "1-3"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "SplitBySizeOrCount", path: "/api/v1/general/split-by-size-or-count",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.SplitBySizeOrCount(ctx, testPDF("a.pdf"), SplitSizeOptions{SplitType: SplitByPageCount, SplitValue: "5"})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"splitType": "1", "splitValue": "5"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "ScalePages", path: "/api/v1/general/scale-pages",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.ScalePages(ctx, testPDF("a.pdf"), ScaleOptions{PageSize: PageSizeA4, ScaleFactor: 0.5})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"pageSize": "A4", "scaleFactor": "0.5"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "Rotate", path: "/api/v1/general/rotate-pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.Rotate(ctx, testPDF("a.pdf"), 90)
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"angle": "90"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "RemovePages", path: "/api/v1/general/remove-pages",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.RemovePages(ctx, testPDF("a.pdf"), "2,4")
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"pageNumbers": "2,4"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "RemoveImages", path: "/api/v1/general/remove-image-pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.RemoveImages(ctx, testPDF("a.pdf"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "RearrangePages", path: "/api/v1/general/rearrange-pages",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.RearrangePages(ctx, testPDF("a.pdf"), RearrangeOptions{Pages: "3,1,2"})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"pageNumbers": "3,1,2"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "ToSinglePage", path: "/api/v1/general/pdf-to-single-page",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.ToSinglePage(ctx, testPDF("a.pdf"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "OverlayPDFs", path: "/api/v1/general/overlay-pdfs",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.OverlayPDFs(ctx, testPDF("base.pdf"), []FileInput{testPDF("overlay.pdf")}, OverlayOptions{Mode: OverlaySequential, Position: OverlayBackground, Counts: []int{2}})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want:  map[string]string{"overlayMode": "SequentialOverlay", "overlayPosition": "1", "counts": "2"},
			files: map[string][]string{"fileInput": {"base.pdf"}, "overlayFiles": {"overlay.pdf"}},
		},
		{
			name: "MultiPageLayout", path: "/api/v1/general/multi-page-layout",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.MultiPageLayout(ctx, testPDF("a.pdf"), MultiPageOptions{PagesPerSheet: 4})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"pagesPerSheet": "4", "addBorder": "false"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "Merge", path: "/api/v1/general/merge-pdfs",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.Merge(ctx, []FileInput{testPDF("a.pdf"), testPDF("b.pdf")}, &MergeOptions{SortType: SortOrderFileName, GenerateTOC: Ptr(true)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want:  map[string]string{"sortType": "byFileName", "generateToc": "true"},
			files: map[string][]string{"fileInput": {"a.pdf", "b.pdf"}},
		},
		{
			name: "ExtractBookmarks", path: "/api/v1/general/extract-bookmarks",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.ExtractBookmarks(ctx, testPDF("a.pdf"))
				return err
			},
			files: map[string][]string{"file": {"a.pdf"}},
		},
		{
			name: "EditTableOfContents", path: "/api/v1/general/edit-table-of-contents",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.EditTableOfContents(ctx, testPDF("a.pdf"), `{"bookmarks":[]}`, true)
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"bookmarkData": `{"bookmarks":[]}`, "replaceExisting": "true"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "Crop", path: "/api/v1/general/crop",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.Crop(ctx, testPDF("a.pdf"), CropOptions{X: 1, Y: 2, Width: 3, Height: 4})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"x": "1", "y": "2", "width": "3", "height": "4"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "Booklet", path: "/api/v1/general/booklet-imposition",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.Booklet(ctx, testPDF("a.pdf"), BookletOptions{Type: BookletStandard, PagesPerSheet: 2, Orientation: Landscape})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"bookletType": "BOOKLET", "pagesPerSheet": "2", "pageOrientation": "LANDSCAPE"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "UpdateMetadata", path: "/api/v1/misc/update-metadata",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.UpdateMetadata(ctx, testPDF("a.pdf"), MetadataOptions{Title: "Hello", Author: "Marcel", Custom: map[string]string{"customKey1": "v"}})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want:  map[string]string{"title": "Hello", "author": "Marcel", "allRequestParams[customKey1]": "v", "deleteAll": "false"},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "UnlockForms", path: "/api/v1/misc/unlock-pdf-forms",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.UnlockForms(ctx, testPDF("a.pdf"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "ShowJavaScript", path: "/api/v1/misc/show-javascript",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.ShowJavaScript(ctx, testPDF("a.pdf"))
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "ScannerEffect", path: "/api/v1/misc/scanner-effect",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.ScannerEffect(ctx, testPDF("a.pdf"), ScannerEffectOptions{Quality: ScanQualityHigh, Rotation: ScanRotationSlight, Colorspace: ColorModeColor})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"quality": "high", "rotation": "slight", "colorspace": "color"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "ReplaceInvertColors", path: "/api/v1/misc/replace-invert-pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.ReplaceInvertColors(ctx, testPDF("a.pdf"), ReplaceInvertColorsOptions{Option: ReplaceFullInversion})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"replaceAndInvertOption": "FULL_INVERSION"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "Repair", path: "/api/v1/misc/repair",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.Repair(ctx, testPDF("a.pdf"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "RemoveBlankPages", path: "/api/v1/misc/remove-blanks",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.RemoveBlankPages(ctx, testPDF("a.pdf"), RemoveBlanksOptions{Threshold: 10, WhitePercent: 99.5})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"threshold": "10", "whitePercent": "99.5"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "OCR", path: "/api/v1/misc/ocr-pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.OCR(ctx, testPDF("a.pdf"), OCROptions{Languages: []string{"eng", "deu"}, Type: OCRForce, RenderType: OCRHOCR})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"ocrType": "force-ocr", "ocrRenderType": "hocr"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "Flatten", path: "/api/v1/misc/flatten",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.Flatten(ctx, testPDF("a.pdf"), true)
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"flattenOnlyForms": "true"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "ExtractImages", path: "/api/v1/misc/extract-images",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.ExtractImages(ctx, testPDF("a.pdf"), ExtractImagesOptions{Format: ImagePNG})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"format": "png"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "ExtractImageScans", path: "/api/v1/misc/extract-image-scans",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.ExtractImageScans(ctx, testPDF("a.pdf"), ExtractImageScansOptions{AngleThreshold: 5, Tolerance: 3, MinArea: 100, MinContourArea: 50, BorderSize: 2})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want:  map[string]string{"angleThreshold": "5", "tolerance": "3", "minArea": "100", "minContourArea": "50", "borderSize": "2"},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "Decompress", path: "/api/v1/misc/decompress-pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.Decompress(ctx, testPDF("a.pdf"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "Compress", path: "/api/v1/misc/compress-pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.Compress(ctx, testPDF("a.pdf"), &CompressOptions{OptimizeLevel: Ptr(5), ExpectedOutputSize: "100KB", Grayscale: Ptr(true)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"optimizeLevel": "5", "expectedOutputSize": "100KB", "grayscale": "true"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "AutoSplit", path: "/api/v1/misc/auto-split-pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.AutoSplit(ctx, testPDF("a.pdf"), &AutoSplitOptions{DuplexMode: Ptr(true)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"duplexMode": "true"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "AutoRename", path: "/api/v1/misc/auto-rename",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.AutoRename(ctx, testPDF("a.pdf"), nil)
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "AddStamp", path: "/api/v1/misc/add-stamp",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.AddStamp(ctx, testPDF("a.pdf"), StampOptions{Type: StampText, Text: "Approved", Position: Ptr(PositionTopRight)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"stampType": "text", "stampText": "Approved", "position": "3"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "AddPageNumbers", path: "/api/v1/misc/add-page-numbers",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.AddPageNumbers(ctx, testPDF("a.pdf"), PageNumbersOptions{Position: Ptr(PositionBottomCenter), FontType: FontHelvetica})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"position": "8", "fontType": "helvetica"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "AddImage", path: "/api/v1/misc/add-image",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.AddImage(ctx, testPDF("a.pdf"), AddImageOptions{Image: testPDF("logo.png"), X: 5, Y: 6, EveryPage: true})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want:  map[string]string{"x": "5", "y": "6", "everyPage": "true"},
			files: map[string][]string{"fileInput": {"a.pdf"}, "imageFile": {"logo.png"}},
		},
		{
			name: "AddAttachments", path: "/api/v1/misc/add-attachments",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.AddAttachments(ctx, testPDF("a.pdf"), []FileInput{FromString("notes.txt", "hi")})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}, "attachments": {"notes.txt"}},
		},
		{
			name: "ConvertURL", path: "/api/v1/convert/url/pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.ConvertURL(ctx, "https://example.com")
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"urlInput": "https://example.com"},
		},
		{
			name: "PDFToXML", path: "/api/v1/convert/pdf/xml",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.PDFToXML(ctx, testPDF("a.pdf"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "PDFToWord", path: "/api/v1/convert/pdf/word",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.PDFToWord(ctx, testPDF("a.pdf"), PDFToDOCX)
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"outputFormat": "docx"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "PDFToText", path: "/api/v1/convert/pdf/text",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.PDFToText(ctx, testPDF("a.pdf"), PDFToTXT)
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"outputFormat": "txt"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "PDFToPresentation", path: "/api/v1/convert/pdf/presentation",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.PDFToPresentation(ctx, testPDF("a.pdf"), PDFToPPTX)
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"outputFormat": "pptx"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "PDFToPDFA", path: "/api/v1/convert/pdf/pdfa",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.PDFToPDFA(ctx, testPDF("a.pdf"), PDFA)
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"outputFormat": "pdfa"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "PDFToMarkdown", path: "/api/v1/convert/pdf/markdown",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.PDFToMarkdown(ctx, testPDF("a.pdf"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "PDFToImages", path: "/api/v1/convert/pdf/img",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.PDFToImages(ctx, testPDF("a.pdf"), PDFToImagesOptions{Format: ImagePNG, Output: ImageSingle, ColorType: ColorModeColor, DPI: Ptr(150)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want:  map[string]string{"imageFormat": "png", "singleOrMultiple": "single", "colorType": "color", "dpi": "150"},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "PDFToHTML", path: "/api/v1/convert/pdf/html",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.PDFToHTML(ctx, testPDF("a.pdf"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "PDFToCSV", path: "/api/v1/convert/pdf/csv",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.PDFToCSV(ctx, testPDF("a.pdf"), "1-2")
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"pageNumbers": "1-2"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "MarkdownToPDF", path: "/api/v1/convert/markdown/pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.MarkdownToPDF(ctx, FromString("a.md", "# hi"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.md"}},
		},
		{
			name: "ImagesToPDF", path: "/api/v1/convert/img/pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.ImagesToPDF(ctx, []FileInput{testPDF("a.png"), testPDF("b.png")}, ImagesToPDFOptions{Fit: ImageFitFillPage, ColorType: ColorModeColor})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want:  map[string]string{"fitOption": "fillPage", "colorType": "color"},
			files: map[string][]string{"fileInput": {"a.png", "b.png"}},
		},
		{
			name: "HTMLToPDF", path: "/api/v1/convert/html/pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.HTMLToPDF(ctx, FromString("a.html", "<h1>hi</h1>"), &HTMLToPDFOptions{Zoom: Ptr(1.5)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"zoom": "1.5"}, files: map[string][]string{"fileInput": {"a.html"}},
		},
		{
			name: "FileToPDF", path: "/api/v1/convert/file/pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.FileToPDF(ctx, FromString("a.txt", "hi"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.txt"}},
		},
		{
			name: "EMLToPDF", path: "/api/v1/convert/eml/pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.EMLToPDF(ctx, FromString("a.eml", "mail"), &EMLToPDFOptions{IncludeAttachments: Ptr(true)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"includeAttachments": "true"}, files: map[string][]string{"fileInput": {"a.eml"}},
		},
		{
			name: "ValidateSignature", path: "/api/v1/security/validate-signature",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.ValidateSignature(ctx, testPDF("a.pdf"), &SignatureValidationOptions{CertFile: testPDF("cert.pem")})
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}, "certFile": {"cert.pem"}},
		},
		{
			name: "Sanitize", path: "/api/v1/security/sanitize-pdf",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.Sanitize(ctx, testPDF("a.pdf"), &SanitizeOptions{RemoveJavaScript: Ptr(true)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"removeJavaScript": "true"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "RemovePassword", path: "/api/v1/security/remove-password",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.RemovePassword(ctx, testPDF("a.pdf"), "secret")
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"password": "secret"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "RemoveCertSign", path: "/api/v1/security/remove-cert-sign",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.RemoveCertSign(ctx, testPDF("a.pdf"))
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "RedactManual", path: "/api/v1/security/redact",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.RedactManual(ctx, testPDF("a.pdf"), RedactManualOptions{
					Pages:          "1",
					RedactionColor: "#000000",
					Redactions:     []RedactionArea{{X: 1, Y: 2, Width: 3, Height: 4, Page: 1}},
				})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want:  map[string]string{"pageNumbers": "1", "pageRedactionColor": "#000000", "redactions[0][x]": "1", "redactions[0][page]": "1"},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "PDFInfo", path: "/api/v1/security/get-info-on-pdf",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.PDFInfo(ctx, testPDF("a.pdf"))
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "SignPDF", path: "/api/v1/security/cert-sign",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.SignPDF(ctx, testPDF("a.pdf"), SignPDFOptions{CertType: CertPKCS12, P12File: testPDF("cert.p12"), Password: "pw"})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want:  map[string]string{"certType": "PKCS12", "password": "pw"},
			files: map[string][]string{"fileInput": {"a.pdf"}, "p12File": {"cert.p12"}},
		},
		{
			name: "RedactAuto", path: "/api/v1/security/auto-redact",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.RedactAuto(ctx, testPDF("a.pdf"), "secret,top-secret", RedactAutoOptions{UseRegex: Ptr(true)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"listOfText": "secret,top-secret", "useRegex": "true"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "AddWatermark", path: "/api/v1/security/add-watermark",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.AddWatermark(ctx, testPDF("a.pdf"), WatermarkOptions{Type: WatermarkText, Text: "Draft", Opacity: Ptr(0.3)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want:  map[string]string{"watermarkType": "text", "watermarkText": "Draft", "opacity": "0.3"},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "AddPassword", path: "/api/v1/security/add-password",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.AddPassword(ctx, testPDF("a.pdf"), PasswordOptions{UserPassword: "u", OwnerPassword: "o", KeyLength: Ptr(256)})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"password": "u", "ownerPassword": "o", "keyLength": "256"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "FilterPageSize", path: "/api/v1/filter/filter-page-size",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.FilterPageSize(ctx, testPDF("a.pdf"), FilterPageSizeOptions{Comparator: Greater, PageSize: PageSizeA4})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"comparator": "Greater", "standardPageSize": "A4"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "FilterPageRotation", path: "/api/v1/filter/filter-page-rotation",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.FilterPageRotation(ctx, testPDF("a.pdf"), FilterPageRotationOptions{Comparator: Equal, Rotation: 90})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"comparator": "Equal", "rotation": "90"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "FilterPageCount", path: "/api/v1/filter/filter-page-count",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.FilterPageCount(ctx, testPDF("a.pdf"), FilterPageCountOptions{Comparator: Less, Count: 10})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"comparator": "Less", "pageCount": "10"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "FilterFileSize", path: "/api/v1/filter/filter-file-size",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.FilterFileSize(ctx, testPDF("a.pdf"), FilterFileSizeOptions{Comparator: Greater, Size: 1024})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"comparator": "Greater", "fileSize": "1024"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "FilterContainsText", path: "/api/v1/filter/filter-contains-text",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.FilterContainsText(ctx, testPDF("a.pdf"), FilterContainsTextOptions{Pages: "1", Text: "invoice"})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"pageNumbers": "1", "text": "invoice"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "FilterContainsImage", path: "/api/v1/filter/filter-contains-image",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.FilterContainsImage(ctx, testPDF("a.pdf"), FilterContainsImageOptions{Pages: "all"})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			want: map[string]string{"pageNumbers": "all"}, files: map[string][]string{"fileInput": {"a.pdf"}},
		},
		{
			name: "RunPipeline", path: "/api/v1/pipeline/handleData",
			call: func(ctx context.Context, c *Client) error {
				res, err := c.RunPipeline(ctx, []FileInput{testPDF("a.pdf")}, Pipeline{Name: "test", Steps: []PipelineStep{{Operation: "/api/v1/misc/repair"}}})
				if err == nil {
					_ = res.Close()
				}
				return err
			},
			files: map[string][]string{"fileInput": {"a.pdf"}},
		},
	}

	server := newCaptureServer(t)
	client, err := New(WithBaseURL(server.server.URL), WithRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server.mu.Lock()
			server.requests = nil
			server.mu.Unlock()
			if err := tc.call(context.Background(), client); err != nil {
				t.Fatalf("call: %v", err)
			}
			got := server.last(t)
			if got.Path != tc.path {
				t.Fatalf("path = %q, want %q", got.Path, tc.path)
			}
			for name, want := range tc.want {
				values := got.Values[name]
				if len(values) == 0 || values[0] != want {
					t.Errorf("form[%q] = %v, want %q", name, values, want)
				}
			}
			for name, want := range tc.files {
				files := got.Files[name]
				var names []string
				for _, file := range files {
					names = append(names, file.Filename)
				}
				if strings.Join(names, ",") != strings.Join(want, ",") {
					t.Errorf("files[%q] = %v, want %v", name, names, want)
				}
			}
		})
	}
}

func TestAnalysisPaths(t *testing.T) {
	server := newCaptureServer(t)
	client, _ := New(WithBaseURL(server.server.URL), WithRetries(0))
	calls := []struct {
		path string
		call func() error
	}{
		{"/api/v1/analysis/security-info", func() error { _, err := client.SecurityInfo(context.Background(), testPDF("a.pdf")); return err }},
		{"/api/v1/analysis/page-dimensions", func() error { _, err := client.PageDimensions(context.Background(), testPDF("a.pdf")); return err }},
		{"/api/v1/analysis/page-count", func() error { _, err := client.PageCount(context.Background(), testPDF("a.pdf")); return err }},
		{"/api/v1/analysis/form-fields", func() error { _, err := client.FormFields(context.Background(), testPDF("a.pdf")); return err }},
		{"/api/v1/analysis/font-info", func() error { _, err := client.FontInfo(context.Background(), testPDF("a.pdf")); return err }},
		{"/api/v1/analysis/document-properties", func() error { _, err := client.DocumentProperties(context.Background(), testPDF("a.pdf")); return err }},
		{"/api/v1/analysis/basic-info", func() error { _, err := client.BasicInfo(context.Background(), testPDF("a.pdf")); return err }},
		{"/api/v1/analysis/annotation-info", func() error { _, err := client.AnnotationInfo(context.Background(), testPDF("a.pdf")); return err }},
	}
	for _, c := range calls {
		if err := c.call(); err != nil {
			t.Fatalf("%s: %v", c.path, err)
		}
		if got := server.last(t).Path; got != c.path {
			t.Errorf("path = %q, want %q", got, c.path)
		}
	}
}

func TestStatus(t *testing.T) {
	server := newCaptureServer(t)
	client, _ := New(WithBaseURL(server.server.URL))
	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Version != "3.3.0" || status.Status != "UP" {
		t.Fatalf("status = %+v", status)
	}
	if got := server.last(t).Method; got != http.MethodGet {
		t.Fatalf("method = %q", got)
	}
}

func TestFilterNotMatched(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	client, _ := New(WithBaseURL(srv.URL))
	res, err := client.FilterPageCount(context.Background(), testPDF("a.pdf"), FilterPageCountOptions{Comparator: Greater, Count: 100})
	if err != nil {
		t.Fatal(err)
	}
	if res.Matched || res.Result != nil {
		t.Fatalf("result = %+v", res)
	}
}

func TestFileIDInput(t *testing.T) {
	server := newCaptureServer(t)
	client, _ := New(WithBaseURL(server.server.URL))
	res, err := client.Rotate(context.Background(), FromFileID("file-1"), 90)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Close()
	got := server.last(t)
	if values := got.Values["fileId"]; len(values) != 1 || values[0] != "file-1" {
		t.Fatalf("fileId = %v", values)
	}
	if len(got.Files["fileInput"]) != 0 {
		t.Fatal("unexpected file upload")
	}
}

func TestRequiredInputs(t *testing.T) {
	client, _ := New(WithBaseURL("http://127.0.0.1:1"))
	if _, err := client.Rotate(context.Background(), nil, 90); err == nil {
		t.Error("Rotate(nil) should fail")
	}
	if _, err := client.Merge(context.Background(), nil, nil); err == nil {
		t.Error("Merge(nil) should fail")
	}
	if _, err := client.SplitPages(context.Background(), testPDF("a.pdf"), ""); err == nil {
		t.Error("SplitPages(empty) should fail")
	}
}
