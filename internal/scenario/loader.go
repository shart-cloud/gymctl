package scenario

import (
	"fmt"
	"os"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"
)

func LoadExerciseFile(path string) (*Exercise, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read exercise file: %w", err)
	}

	if err := ValidateExerciseYAML(data); err != nil {
		return nil, err
	}

	var exercise Exercise
	if err := yaml.Unmarshal(data, &exercise); err != nil {
		return nil, fmt.Errorf("parse exercise yaml: %w", err)
	}

	return &exercise, nil
}

func ValidateExerciseYAML(data []byte) error {
	jsonBytes, err := yaml.YAMLToJSON(data)
	if err != nil {
		return fmt.Errorf("convert yaml to json: %w", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(exerciseSchema)
	documentLoader := gojsonschema.NewBytesLoader(jsonBytes)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("validate schema: %w", err)
	}

	if result.Valid() {
		return nil
	}

	var messages []string
	for _, err := range result.Errors() {
		messages = append(messages, err.String())
	}

	return fmt.Errorf("exercise schema validation failed: %s", strings.Join(messages, "; "))
}
