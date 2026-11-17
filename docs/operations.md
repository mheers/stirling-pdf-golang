# Operations

Generated from `api/openapi.json` by `go generate ./...`. Every processing operation is available
as `stirling-pdf <group> <command>` and through the matching SDK method.

## General

| Command | Path | Summary |
|---|---|---|
| `general booklet-imposition` | `/api/v1/general/booklet-imposition` | Create a booklet with proper page imposition |
| `general crop` | `/api/v1/general/crop` | Crops a PDF document |
| `general edit-table-of-contents` | `/api/v1/general/edit-table-of-contents` | Edit Table of Contents |
| `general extract-bookmarks` | `/api/v1/general/extract-bookmarks` | Extract PDF Bookmarks |
| `general merge-pdfs` | `/api/v1/general/merge-pdfs` | Merge multiple PDF files into one |
| `general multi-page-layout` | `/api/v1/general/multi-page-layout` | Merge multiple pages of a PDF document into a single page |
| `general overlay-pdfs` | `/api/v1/general/overlay-pdfs` | Overlay PDF files in various modes |
| `general pdf-to-single-page` | `/api/v1/general/pdf-to-single-page` | Convert a multi-page PDF into a single long page PDF |
| `general rearrange-pages` | `/api/v1/general/rearrange-pages` | Rearrange pages in a PDF file |
| `general remove-image-pdf` | `/api/v1/general/remove-image-pdf` | Remove images from file to reduce the file size. |
| `general remove-pages` | `/api/v1/general/remove-pages` | Remove pages from a PDF file |
| `general rotate-pdf` | `/api/v1/general/rotate-pdf` | Rotate a PDF file |
| `general scale-pages` | `/api/v1/general/scale-pages` | Change the size of a PDF page/document |
| `general split-by-size-or-count` | `/api/v1/general/split-by-size-or-count` | Auto split PDF pages into separate documents based on size or count |
| `general split-pages` | `/api/v1/general/split-pages` | Split a PDF file into separate documents |
| `general split-pdf-by-chapters` | `/api/v1/general/split-pdf-by-chapters` | Split PDFs by Chapters |
| `general split-pdf-by-sections` | `/api/v1/general/split-pdf-by-sections` | Split PDF pages into smaller sections |

## Convert

| Command | Path | Summary |
|---|---|---|
| `convert eml-pdf` | `/api/v1/convert/eml/pdf` | Convert EML to PDF |
| `convert file-pdf` | `/api/v1/convert/file/pdf` | Convert a file to a PDF using LibreOffice |
| `convert html-pdf` | `/api/v1/convert/html/pdf` | Convert an HTML or ZIP (containing HTML and CSS) to PDF |
| `convert img-pdf` | `/api/v1/convert/img/pdf` | Convert images to a PDF file |
| `convert markdown-pdf` | `/api/v1/convert/markdown/pdf` | Convert a Markdown file to PDF |
| `convert pdf-csv` | `/api/v1/convert/pdf/csv` | Extracts a CSV document from a PDF |
| `convert pdf-html` | `/api/v1/convert/pdf/html` | Convert PDF to HTML |
| `convert pdf-img` | `/api/v1/convert/pdf/img` | Convert PDF to image(s) |
| `convert pdf-markdown` | `/api/v1/convert/pdf/markdown` | Convert PDF to Markdown |
| `convert pdf-pdfa` | `/api/v1/convert/pdf/pdfa` | Convert a PDF to a PDF/A |
| `convert pdf-presentation` | `/api/v1/convert/pdf/presentation` | Convert PDF to Presentation format |
| `convert pdf-text` | `/api/v1/convert/pdf/text` | Convert PDF to Text or RTF format |
| `convert pdf-word` | `/api/v1/convert/pdf/word` | Convert PDF to Word document |
| `convert pdf-xml` | `/api/v1/convert/pdf/xml` | Convert PDF to XML |
| `convert url-pdf` | `/api/v1/convert/url/pdf` | Convert a URL to a PDF |

## Security

| Command | Path | Summary |
|---|---|---|
| `security add-password` | `/api/v1/security/add-password` | Add password to a PDF file |
| `security add-watermark` | `/api/v1/security/add-watermark` | Add watermark to a PDF file |
| `security auto-redact` | `/api/v1/security/auto-redact` | Redacts listOfText in a PDF document |
| `security cert-sign` | `/api/v1/security/cert-sign` | Sign PDF with a Digital Certificate |
| `security get-info-on-pdf` | `/api/v1/security/get-info-on-pdf` | Summary here |
| `security redact` | `/api/v1/security/redact` | Redacts areas and pages in a PDF document |
| `security remove-cert-sign` | `/api/v1/security/remove-cert-sign` | Remove digital signature from PDF |
| `security remove-password` | `/api/v1/security/remove-password` | Remove password from a PDF file |
| `security sanitize-pdf` | `/api/v1/security/sanitize-pdf` | Sanitize a PDF file |
| `security validate-signature` | `/api/v1/security/validate-signature` | Validate PDF Digital Signature |

## Misc

| Command | Path | Summary |
|---|---|---|
| `misc add-attachments` | `/api/v1/misc/add-attachments` | Add attachments to PDF |
| `misc add-image` | `/api/v1/misc/add-image` | Overlay image onto a PDF file |
| `misc add-page-numbers` | `/api/v1/misc/add-page-numbers` | Add page numbers to a PDF document |
| `misc add-stamp` | `/api/v1/misc/add-stamp` | Add stamp to a PDF file |
| `misc auto-rename` | `/api/v1/misc/auto-rename` | Extract header from PDF file |
| `misc auto-split-pdf` | `/api/v1/misc/auto-split-pdf` | Auto split PDF pages into separate documents |
| `misc compress-pdf` | `/api/v1/misc/compress-pdf` | Optimize PDF file |
| `misc decompress-pdf` | `/api/v1/misc/decompress-pdf` | Decompress PDF streams |
| `misc extract-image-scans` | `/api/v1/misc/extract-image-scans` | Extract image scans from an input file |
| `misc extract-images` | `/api/v1/misc/extract-images` | Extract images from a PDF file |
| `misc flatten` | `/api/v1/misc/flatten` | Flatten PDF form fields or full page |
| `misc ocr-pdf` | `/api/v1/misc/ocr-pdf` | Process a PDF file with OCR |
| `misc remove-blanks` | `/api/v1/misc/remove-blanks` | Remove blank pages from a PDF file |
| `misc repair` | `/api/v1/misc/repair` | Repair a PDF file |
| `misc replace-invert-pdf` | `/api/v1/misc/replace-invert-pdf` | Replace-Invert Color PDF |
| `misc scanner-effect` | `/api/v1/misc/scanner-effect` | Apply scanner effect to PDF |
| `misc show-javascript` | `/api/v1/misc/show-javascript` | Grabs all JS from a PDF and returns a single JS file with all code |
| `misc unlock-pdf-forms` | `/api/v1/misc/unlock-pdf-forms` | Remove read-only property from form fields |
| `misc update-metadata` | `/api/v1/misc/update-metadata` | Update metadata of a PDF file |

## Analysis

| Command | Path | Summary |
|---|---|---|
| `analysis annotation-info` | `/api/v1/analysis/annotation-info` | Get annotation information |
| `analysis basic-info` | `/api/v1/analysis/basic-info` | Get basic PDF information |
| `analysis document-properties` | `/api/v1/analysis/document-properties` | Get PDF document properties |
| `analysis font-info` | `/api/v1/analysis/font-info` | Get font information |
| `analysis form-fields` | `/api/v1/analysis/form-fields` | Get form field information |
| `analysis page-count` | `/api/v1/analysis/page-count` | Get PDF page count |
| `analysis page-dimensions` | `/api/v1/analysis/page-dimensions` | Get page dimensions for all pages |
| `analysis security-info` | `/api/v1/analysis/security-info` | Get security information |

## Filter

| Command | Path | Summary |
|---|---|---|
| `filter filter-contains-image` | `/api/v1/filter/filter-contains-image` | Checks if a PDF contains an image |
| `filter filter-contains-text` | `/api/v1/filter/filter-contains-text` | Checks if a PDF contains set text, returns true if does |
| `filter filter-file-size` | `/api/v1/filter/filter-file-size` | Checks if a PDF is a set file size |
| `filter filter-page-count` | `/api/v1/filter/filter-page-count` | Checks if a PDF is greater, less or equal to a setPageCount |
| `filter filter-page-rotation` | `/api/v1/filter/filter-page-rotation` | Checks if a PDF is of a certain rotation |
| `filter filter-page-size` | `/api/v1/filter/filter-page-size` | Checks if a PDF is of a certain size |

## Pipeline

| Command | Path | Summary |
|---|---|---|
| `pipeline handle-data` | `/api/v1/pipeline/handleData` | Execute automated PDF processing pipeline |
