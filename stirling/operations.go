package stirling

import (
	"fmt"
)

func inputParams(in FileInput) ([]Param, error) {
	if isNilFile(in) {
		return nil, fmt.Errorf("stirling: an input file is required")
	}
	if id, ok := FileID(in); ok {
		return []Param{{Name: "fileId", Value: id}}, nil
	}
	return []Param{{Name: "fileInput", Value: in}}, nil
}

func multiInputParams(name string, files []FileInput) ([]Param, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("stirling: at least one input file is required")
	}
	for i, file := range files {
		if isNilFile(file) {
			return nil, fmt.Errorf("stirling: input file %d is nil", i)
		}
		if _, ok := FileID(file); ok {
			return nil, fmt.Errorf("stirling: server-side file references are not supported for %s", name)
		}
	}
	return []Param{{Name: name, Value: files}}, nil
}
