package helpers

// FieldFormatter функция форматирования значения поля
type FieldFormatter func(value any) (string, string)

// Field представляет поле с именем, значением и единицей измерения
type Field struct {
	Name      string
	Value     any
	Unit      string
	Formatter FieldFormatter
}
