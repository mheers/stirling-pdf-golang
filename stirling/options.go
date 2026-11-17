package stirling

// Ptr returns a pointer to v. It is a shorthand for option fields such as
// MergeOptions{GenerateTOC: stirling.Ptr(true)}.
func Ptr[T any](v T) *T { return &v }

// SortType controls the order in which Merge sorts input files.
type SortType string

// Sort strategies supported by the merge endpoint.
const (
	SortOrderProvided   SortType = "orderProvided"
	SortOrderFileName   SortType = "byFileName"
	SortOrderDateMod    SortType = "byDateModified"
	SortOrderDateCreate SortType = "byDateCreated"
	SortOrderPDFTitle   SortType = "byPDFTitle"
)

// PageSize is a target paper size for ScalePages.
type PageSize string

// Paper sizes supported by ScalePages.
const (
	PageSizeA0     PageSize = "A0"
	PageSizeA1     PageSize = "A1"
	PageSizeA2     PageSize = "A2"
	PageSizeA3     PageSize = "A3"
	PageSizeA4     PageSize = "A4"
	PageSizeA5     PageSize = "A5"
	PageSizeA6     PageSize = "A6"
	PageSizeLetter PageSize = "LETTER"
	PageSizeLegal  PageSize = "LEGAL"
	PageSizeKeep   PageSize = "KEEP"
)

// OverlayMode controls how OverlayPDFs applies overlay documents.
type OverlayMode string

// Overlay strategies supported by the overlay endpoint.
const (
	OverlaySequential  OverlayMode = "SequentialOverlay"
	OverlayInterleaved OverlayMode = "InterleavedOverlay"
	OverlayFixedRepeat OverlayMode = "FixedRepeatOverlay"
)

// OverlayPosition selects whether overlays are drawn in front of or behind the
// base document.
type OverlayPosition int

// Overlay positions supported by the overlay endpoint.
const (
	OverlayForeground OverlayPosition = 0
	OverlayBackground OverlayPosition = 1
)

// SplitType selects the criterion for SplitBySizeOrCount.
type SplitType int

// Split criteria supported by SplitBySizeOrCount.
const (
	SplitBySize      SplitType = 0
	SplitByPageCount SplitType = 1
	SplitByDocCount  SplitType = 2
)

// RearrangeMode selects a built-in page ordering for RearrangePages.
type RearrangeMode string

// Page ordering modes supported by RearrangePages. Use CustomRearrangeMode
// together with RearrangeOptions.Pages for an explicit order.
const (
	RearrangeCustom            RearrangeMode = "CUSTOM"
	RearrangeReverse           RearrangeMode = "REVERSE_ORDER"
	RearrangeDuplex            RearrangeMode = "DUPLEX_SORT"
	RearrangeBooklet           RearrangeMode = "BOOKLET_SORT"
	RearrangeSideStitchBooklet RearrangeMode = "SIDE_STITCH_BOOKLET_SORT"
	RearrangeOddEvenSplit      RearrangeMode = "ODD_EVEN_SPLIT"
	RearrangeOddEvenMerge      RearrangeMode = "ODD_EVEN_MERGE"
	RearrangeRemoveFirst       RearrangeMode = "REMOVE_FIRST"
	RearrangeRemoveLast        RearrangeMode = "REMOVE_LAST"
	RearrangeRemoveFirstLast   RearrangeMode = "REMOVE_FIRST_AND_LAST"
	RearrangeDuplicate         RearrangeMode = "DUPLICATE"
)

// BookletType selects the binding style for Booklet.
type BookletType string

// Booklet binding styles.
const (
	BookletStandard   BookletType = "BOOKLET"
	BookletSideStitch BookletType = "SIDE_STITCH_BOOKLET"
)

// Orientation is a page orientation.
type Orientation string

// Page orientations.
const (
	Portrait  Orientation = "PORTRAIT"
	Landscape Orientation = "LANDSCAPE"
)

// WatermarkType selects a text or image watermark.
type WatermarkType string

// Watermark types.
const (
	WatermarkText  WatermarkType = "text"
	WatermarkImage WatermarkType = "image"
)

// StampType selects a text or image stamp.
type StampType string

// Stamp types.
const (
	StampText  StampType = "text"
	StampImage StampType = "image"
)

// Alphabet selects the glyph set for text watermarks, stamps, and page
// numbers.
type Alphabet string

// Supported alphabets.
const (
	AlphabetRoman    Alphabet = "roman"
	AlphabetArabic   Alphabet = "arabic"
	AlphabetJapanese Alphabet = "japanese"
	AlphabetKorean   Alphabet = "korean"
	AlphabetChinese  Alphabet = "chinese"
)

// FontType selects a built-in font for page numbers.
type FontType string

// Supported page-number fonts.
const (
	FontCourier   FontType = "courier"
	FontHelvetica FontType = "helvetica"
	FontTimes     FontType = "times"
)

// Margin selects a predefined page margin.
type Margin string

// Predefined margins.
const (
	MarginSmall  Margin = "small"
	MarginMedium Margin = "medium"
	MarginLarge  Margin = "large"
	MarginXLarge Margin = "x-large"
)

// Page position identifiers used by stamps, page numbers, and watermarks.
const (
	PositionTopLeft      = 1
	PositionTopCenter    = 2
	PositionTopRight     = 3
	PositionMiddleLeft   = 4
	PositionMiddleCenter = 5
	PositionMiddleRight  = 6
	PositionBottomLeft   = 7
	PositionBottomCenter = 8
	PositionBottomRight  = 9
)

// ImageFormat is an output image format.
type ImageFormat string

// Image formats supported by the conversion endpoints.
const (
	ImagePNG  ImageFormat = "png"
	ImageJPEG ImageFormat = "jpeg"
	ImageJPG  ImageFormat = "jpg"
	ImageGIF  ImageFormat = "gif"
	ImageWebP ImageFormat = "webp"
)

// ColorType controls color conversion for images and scans.
type ColorType string

// Color modes supported by the conversion and scan endpoints.
const (
	ColorModeColor      ColorType = "color"
	ColorModeGreyscale  ColorType = "greyscale"
	ColorModeBlackWhite ColorType = "blackwhite"
)

// ImageOutput selects one combined image or one image per page.
type ImageOutput string

// Image output modes.
const (
	ImageSingle   ImageOutput = "single"
	ImageMultiple ImageOutput = "multiple"
)

// ImageFit controls how images are placed on pages by ImagesToPDF.
type ImageFit string

// Image fit options.
const (
	ImageFitFillPage    ImageFit = "fillPage"
	ImageFitDocument    ImageFit = "fitDocumentToImage"
	ImageFitAspectRatio ImageFit = "maintainAspectRatio"
)

// OCRType controls how the OCR endpoint treats existing text.
type OCRType string

// OCR modes.
const (
	OCRSkipText OCRType = "skip-text"
	OCRForce    OCRType = "force-ocr"
	OCRStandard OCRType = "Normal"
)

// OCRRenderType selects the OCR output format.
type OCRRenderType string

// OCR render types.
const (
	OCRHOCR     OCRRenderType = "hocr"
	OCRSandwich OCRRenderType = "sandwich"
)

// ScanQuality controls the scanner-effect quality.
type ScanQuality string

// Scanner-effect qualities.
const (
	ScanQualityLow    ScanQuality = "low"
	ScanQualityMedium ScanQuality = "medium"
	ScanQualityHigh   ScanQuality = "high"
)

// ScanRotation controls the scanner-effect rotation amount.
type ScanRotation string

// Scanner-effect rotations.
const (
	ScanRotationNone     ScanRotation = "none"
	ScanRotationSlight   ScanRotation = "slight"
	ScanRotationModerate ScanRotation = "moderate"
	ScanRotationSevere   ScanRotation = "severe"
)

// ReplaceColorOption selects a color replacement strategy.
type ReplaceColorOption string

// Color replacement strategies.
const (
	ReplaceHighContrast  ReplaceColorOption = "HIGH_CONTRAST_COLOR"
	ReplaceCustomColor   ReplaceColorOption = "CUSTOM_COLOR"
	ReplaceFullInversion ReplaceColorOption = "FULL_INVERSION"
)

// HighContrastCombination selects a foreground/background pair.
type HighContrastCombination string

// High-contrast color combinations.
const (
	WhiteOnBlack  HighContrastCombination = "WHITE_TEXT_ON_BLACK"
	BlackOnWhite  HighContrastCombination = "BLACK_TEXT_ON_WHITE"
	YellowOnBlack HighContrastCombination = "YELLOW_TEXT_ON_BLACK"
	GreenOnBlack  HighContrastCombination = "GREEN_TEXT_ON_BLACK"
)

// CertType selects the certificate container for SignPDF.
type CertType string

// Certificate container types.
const (
	CertPEM    CertType = "PEM"
	CertPKCS12 CertType = "PKCS12"
	CertJKS    CertType = "JKS"
	CertServer CertType = "SERVER"
)

// Comparator is used by the filter endpoints.
type Comparator string

// Filter comparators.
const (
	Greater Comparator = "Greater"
	Equal   Comparator = "Equal"
	Less    Comparator = "Less"
)

// PDFToWordFormat is an output format for PDFToWord.
type PDFToWordFormat string

// PDF to word processing formats.
const (
	PDFToDOC  PDFToWordFormat = "doc"
	PDFToDOCX PDFToWordFormat = "docx"
	PDFToODT  PDFToWordFormat = "odt"
)

// PDFToTextFormat is an output format for PDFToText.
type PDFToTextFormat string

// PDF to text formats.
const (
	PDFToRTF PDFToTextFormat = "rtf"
	PDFToTXT PDFToTextFormat = "txt"
)

// PDFToPresentationFormat is an output format for PDFToPresentation.
type PDFToPresentationFormat string

// PDF to presentation formats.
const (
	PDFToPPT  PDFToPresentationFormat = "ppt"
	PDFToPPTX PDFToPresentationFormat = "pptx"
	PDFToODP  PDFToPresentationFormat = "odp"
)

// PDFToPDFAFormat is an output format for PDFToPDFA.
type PDFToPDFAFormat string

// PDF/A conversion profiles.
const (
	PDFA  PDFToPDFAFormat = "pdfa"
	PDFA1 PDFToPDFAFormat = "pdfa-1"
)
