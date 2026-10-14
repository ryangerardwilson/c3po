package analyze

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
	"github.com/ryangerardwilson/c3po/complexity/internal/models"
)

type baselineFile struct {
	Functions []metric.Function `json:"functions"`
	Models    []models.Model    `json:"models"`
}

func loadBaseline(path, gitRoot string) ([]metric.Function, []models.Model, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, baselineErr(path, err)
	}
	var file baselineFile
	if err := json.Unmarshal(buf, &file); err != nil {
		return nil, nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return file.Functions, file.Models, nil
	}
	return internalFns(cwd, gitRoot, file.Functions), internalModels(cwd, gitRoot, file.Models), nil
}

func baselineErr(path string, err error) error {
	shown := userPath(path)
	msg := err.Error()
	if path != "" && shown != path {
		msg = strings.ReplaceAll(msg, path, shown)
	}
	return fmt.Errorf("baseline %s: %s", shown, msg)
}

// WriteBaseline stores function and model scores for a later ratchet.
func WriteBaseline(path string, fns []metric.Function, modelsList []models.Model) error {
	buf, err := json.MarshalIndent(baselineFile{Functions: fns, Models: modelsList}, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	return os.WriteFile(path, buf, 0o644)
}
