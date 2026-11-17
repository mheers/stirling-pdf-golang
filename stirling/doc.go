// Package stirling is a Go client for the Stirling PDF Processing API.
//
// The client covers merging, splitting, conversion, security, OCR, analysis,
// filtering, and pipeline operations against both Stirling Cloud and
// self-hosted servers.
//
// Create a client with [New] or [NewFromEnv], upload documents with a
// [FileInput], and save the streamed [Result]:
//
//	client, err := stirling.New(stirling.WithBaseURL("http://localhost:8080"))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	input, err := stirling.FromPath("invoice.pdf")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	result, err := client.Compress(ctx, input, &stirling.CompressOptions{
//		OptimizeLevel: stirling.Ptr(5),
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer result.Close()
//	if err := result.Save("invoice-small.pdf"); err != nil {
//		log.Fatal(err)
//	}
//
// Analysis operations return map[string]any because the API describes their
// JSON responses as free-form objects. Filter operations return a
// [FilterResult] that distinguishes a match from a 204 No Content response.
//
// Low-level access is available through [Client.Call] for endpoints that are
// not wrapped by the typed methods. The generated API lives in the api
// subpackage.
//
// See the repository README and the docs directory for a complete guide.
package stirling
