package formatter

import (
	"fmt"
	"strings"

	"system-stats/internal/types"
)

type Config struct {
	Indent    int
	FieldName string
}

func DefaultConfig() Config {
	return Config{
		Indent:    2,
		FieldName: "  %s%s : %s%s\n",
	}
}

func FormatFields(fields []types.Field, config Config) string {
	if len(fields) == 0 {
		return ""
	}

	maxNameLen := getMaxNameLen(fields)
	indent := strings.Repeat(" ", config.Indent)

	var sb strings.Builder
	sb.WriteString(indent + "─\n")

	for _, field := range fields {
		padding := strings.Repeat(" ", maxNameLen-len(field.Name))
		valueStr := formatValue(field.Value, field.Formatter)
		unitStr := formatUnit(field.Unit)
		sb.WriteString(fmt.Sprintf(config.FieldName, field.Name, padding, valueStr, unitStr))
	}

	sb.WriteString(indent + "─\n")
	return sb.String()
}

func formatValue(value any, formatter types.FieldFormatter) string {
	if formatter != nil {
		formatted, _ := formatter(value)
		return formatted
	}

	switch v := value.(type) {
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatUnit(unit string) string {
	if unit == "" {
		return ""
	}
	return " " + unit
}

func getMaxNameLen(fields []types.Field) int {
	maxLen := 0
	for _, field := range fields {
		if len(field.Name) > maxLen {
			maxLen = len(field.Name)
		}
	}
	return maxLen
}

type Builder struct {
	fields []types.Field
	config Config
}

func NewBuilder() *Builder {
	return &Builder{
		fields: make([]types.Field, 0),
		config: DefaultConfig(),
	}
}

func (b *Builder) WithConfig(config Config) *Builder {
	b.config = config
	return b
}

func (b *Builder) AddField(name string, value any, unit string) *Builder {
	b.fields = append(b.fields, types.Field{
		Name:  name,
		Value: value,
		Unit:  unit,
	})
	return b
}

func (b *Builder) AddFieldWithFormatter(name string, value any, unit string, formatter types.FieldFormatter) *Builder {
	b.fields = append(b.fields, types.Field{
		Name:      name,
		Value:     value,
		Unit:      unit,
		Formatter: formatter,
	})
	return b
}

func (b *Builder) Build() string {
	return FormatFields(b.fields, b.config)
}
