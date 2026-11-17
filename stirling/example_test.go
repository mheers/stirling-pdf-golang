package stirling_test

import (
	"context"
	"log"
	"os"

	"github.com/mheers/stirling-pdf-golang/stirling"
)

func ExampleNew() {
	client, err := stirling.New(
		stirling.WithBaseURL("http://localhost:8080"),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

func ExampleNewFromEnv() {
	// STIRLING_URL, STIRLING_API_KEY, STIRLING_TIMEOUT, and STIRLING_RETRIES
	// are read automatically. Explicit options still win.
	client, err := stirling.NewFromEnv(stirling.WithTimeout(0))
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

func ExampleClient_Merge() {
	client, err := stirling.New(stirling.WithBaseURL("http://localhost:8080"))
	if err != nil {
		log.Fatal(err)
	}

	merged, err := client.Merge(context.Background(), []stirling.FileInput{
		stirling.FromBytes("cover.pdf", []byte("%PDF-1.7 ...")),
		stirling.FromBytes("body.pdf", []byte("%PDF-1.7 ...")),
	}, &stirling.MergeOptions{
		SortType:    stirling.SortOrderFileName,
		GenerateTOC: stirling.Ptr(true),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = merged.Close() }()

	if err := merged.Save("report.pdf"); err != nil {
		log.Fatal(err)
	}
}

func ExampleClient_Compress() {
	client, err := stirling.New(stirling.WithBaseURL("http://localhost:8080"))
	if err != nil {
		log.Fatal(err)
	}

	input, err := stirling.FromPath("invoice.pdf")
	if err != nil {
		log.Fatal(err)
	}
	compressed, err := client.Compress(context.Background(), input, &stirling.CompressOptions{
		OptimizeLevel:      stirling.Ptr(5),
		ExpectedOutputSize: "100KB",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = compressed.Close() }()

	if _, err := compressed.SaveTo(os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func ExampleClient_PageCount() {
	client, err := stirling.New(stirling.WithBaseURL("http://localhost:8080"))
	if err != nil {
		log.Fatal(err)
	}

	var data []byte // contents of report.pdf
	report, err := client.PageCount(context.Background(), stirling.FromBytes("report.pdf", data))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("pages: %v", report["pageCount"])
}

func ExampleClient_FilterContainsText() {
	client, err := stirling.New(stirling.WithBaseURL("http://localhost:8080"))
	if err != nil {
		log.Fatal(err)
	}

	var data []byte // contents of invoice.pdf
	result, err := client.FilterContainsText(context.Background(),
		stirling.FromBytes("invoice.pdf", data),
		stirling.FilterContainsTextOptions{Pages: "1-3", Text: "invoice"},
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = result.Close() }()
	if !result.Matched {
		return
	}
	// result.Result is the original PDF; copy it elsewhere if needed.
}

func ExampleClient_RunPipeline() {
	client, err := stirling.New(stirling.WithBaseURL("http://localhost:8080"))
	if err != nil {
		log.Fatal(err)
	}

	pipeline := stirling.Pipeline{Name: "prepare-for-email"}
	pipeline.Step("/api/v1/security/sanitize-pdf", map[string]any{
		"removeJavaScript": true,
	})
	pipeline.Step("/api/v1/misc/compress-pdf", map[string]any{
		"optimizeLevel": 2,
	})

	var data []byte // contents of input.pdf
	result, err := client.RunPipeline(context.Background(),
		[]stirling.FileInput{stirling.FromBytes("input.pdf", data)},
		pipeline,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = result.Close() }()
}
