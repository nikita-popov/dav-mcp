package mcp

import (
	"errors"
	"fmt"
)

// ArgSchema is a minimal runtime schema used to validate tool arguments
// before they reach handlers. LLMs frequently hallucinate parameters, so
// validating early significantly improves tool reliability.

type ArgSchema struct {
	Required []string
	Optional []string
}

// Validate checks provided arguments against the schema.
// It ensures required fields exist and rejects unknown fields.

func (s ArgSchema) Validate(args map[string]any) error {
	allowed := map[string]struct{}{}

	for _, k := range s.Required {
		allowed[k] = struct{}{}
		if _, ok := args[k]; !ok {
			return fmt.Errorf("missing required argument: %s", k)
		}
	}

	for _, k := range s.Optional {
		allowed[k] = struct{}{}
	}

	for k := range args {
		if _, ok := allowed[k]; !ok {
			return errors.New("unknown argument: " + k)
		}
	}

	return nil
}

// ValidateArgs is a helper wrapper used by tools.
// Example:
//
// schema := ArgSchema{
//     Required: []string{"start","end"},
//     Optional: []string{"calendar"},
// }
//
// if err := ValidateArgs(schema, args); err != nil {
//     return nil, err
// }

func ValidateArgs(schema ArgSchema, args map[string]any) error {
	if args == nil {
		args = map[string]any{}
	}

	return schema.Validate(args)
}
