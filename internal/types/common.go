package types

type FieldFormatter func(value any) (string, string)

type Field struct {
	Name      string
	Value     any
	Unit      string
	Formatter FieldFormatter
}
