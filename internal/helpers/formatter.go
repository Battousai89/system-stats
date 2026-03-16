package helpers

import (
	"fmt"
	"strings"
)

// Config конфигурация форматирования
type Config struct {
	Indent    int
	FieldName string
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		Indent:    2,
		FieldName: "  %s%s : %s%s\n",
	}
}

// FormatFields форматирует список полей
func FormatFields(fields []Field, config Config) string {
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

func formatValue(value any, formatter FieldFormatter) string {
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

func getMaxNameLen(fields []Field) int {
	maxLen := 0
	for _, field := range fields {
		if len(field.Name) > maxLen {
			maxLen = len(field.Name)
		}
	}
	return maxLen
}

// Builder построитель отформатированного вывода
type Builder struct {
	fields []Field
	config Config
}

// NewBuilder создаёт новый Builder
func NewBuilder() *Builder {
	return &Builder{
		fields: make([]Field, 0),
		config: DefaultConfig(),
	}
}

// WithConfig устанавливает конфигурацию
func (b *Builder) WithConfig(config Config) *Builder {
	b.config = config
	return b
}

// AddField добавляет поле
func (b *Builder) AddField(name string, value any, unit string) *Builder {
	b.fields = append(b.fields, Field{
		Name:  name,
		Value: value,
		Unit:  unit,
	})
	return b
}

// AddFieldWithFormatter добавляет поле с форматтером
func (b *Builder) AddFieldWithFormatter(name string, value any, unit string, formatter FieldFormatter) *Builder {
	b.fields = append(b.fields, Field{
		Name:      name,
		Value:     value,
		Unit:      unit,
		Formatter: formatter,
	})
	return b
}

// Build возвращает отформатированную строку
func (b *Builder) Build() string {
	return FormatFields(b.fields, b.config)
}
