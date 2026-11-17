package stirling

import (
	"context"
	"encoding/json"
	"fmt"
)

// PipelineStep is one operation in a [Pipeline]. Operation is a full API path
// such as "/api/v1/misc/compress-pdf".
type PipelineStep struct {
	Operation  string         `json:"operation"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

// Pipeline is the document-processing pipeline accepted by
// [Client.RunPipeline].
type Pipeline struct {
	Name  string         `json:"name,omitempty"`
	Steps []PipelineStep `json:"pipeline"`
}

// Step appends a step and returns the pipeline for chaining.
func (p *Pipeline) Step(operation string, parameters map[string]any) *Pipeline {
	p.Steps = append(p.Steps, PipelineStep{Operation: operation, Parameters: parameters})
	return p
}

// RunPipeline executes several processing operations in one request. The
// server passes the files through the steps in order and returns the final
// result, which may be a PDF or a ZIP archive.
func (c *Client) RunPipeline(ctx context.Context, files []FileInput, pipeline Pipeline) (*Result, error) {
	payload, err := json.Marshal(pipeline)
	if err != nil {
		return nil, fmt.Errorf("stirling: pipeline: encode: %w", err)
	}
	return c.RunPipelineJSON(ctx, files, string(payload))
}

// RunPipelineJSON is [Client.RunPipeline] for a pre-encoded pipeline
// definition.
func (c *Client) RunPipelineJSON(ctx context.Context, files []FileInput, pipelineJSON string) (*Result, error) {
	if pipelineJSON == "" {
		return nil, fmt.Errorf("stirling: pipeline: definition is required")
	}
	params, err := multiInputParams("fileInput", files)
	if err != nil {
		return nil, err
	}
	params = append(params, Param{Name: "json", Value: pipelineJSON})
	return c.Call(ctx, "/api/v1/pipeline/handleData", params...)
}
