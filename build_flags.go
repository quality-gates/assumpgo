package assumpgo

import (
	"fmt"
	"go/build"
	"os"
	"strings"
)

func currentBuildContext() (build.Context, error) {
	context := build.Default
	flags, err := splitGoFlags(os.Getenv("GOFLAGS"))
	if err != nil {
		return context, fmt.Errorf("parsing GOFLAGS: %w", err)
	}
	for _, flag := range flags {
		name, value, hasValue := strings.Cut(flag, "=")
		if name != "-tags" && name != "--tags" {
			continue
		}
		if !hasValue {
			return context, fmt.Errorf("parsing GOFLAGS: %s requires a value", name)
		}
		if strings.ContainsAny(value, " '") {
			context.BuildTags, err = splitGoFlags(value)
			if err != nil {
				return context, fmt.Errorf("parsing GOFLAGS tags: %w", err)
			}
		} else {
			context.BuildTags = strings.FieldsFunc(value, func(r rune) bool { return r == ',' })
		}
	}
	return context, nil
}

func splitGoFlags(s string) ([]string, error) {
	var fields []string
	for len(s) > 0 {
		s = strings.TrimLeft(s, " \t\n\r")
		if s == "" {
			break
		}
		if s[0] == '\'' || s[0] == '"' {
			quote := s[0]
			s = s[1:]
			i := strings.IndexByte(s, quote)
			if i < 0 {
				return nil, fmt.Errorf("unterminated %c string", quote)
			}
			fields = append(fields, s[:i])
			s = s[i+1:]
			continue
		}
		i := strings.IndexAny(s, " \t\n\r")
		if i < 0 {
			fields = append(fields, s)
			break
		}
		fields = append(fields, s[:i])
		s = s[i:]
	}
	return fields, nil
}
