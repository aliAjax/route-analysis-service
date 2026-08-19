package export

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"strings"
)

// ToCSV serialises the report sections into a single CSV document. Each
// section is prefixed by a header row "[section:name]".
func (r Report) ToCSV() ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	for _, section := range r.Sections {
		if err := writer.Write([]string{"[section:" + section.Name + "]"}); err != nil {
			return nil, err
		}
		for _, row := range section.Rows {
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
		if len(section.Totals) > 0 {
			for key, value := range section.Totals {
				if err := writer.Write([]string{key, formatAny(value)}); err != nil {
					return nil, err
				}
			}
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}

// EscapeCell makes an arbitrary string safe for embedding in a single CSV cell.
func EscapeCell(value string) string {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{value})
	writer.Flush()
	out := buffer.String()
	return strings.TrimSuffix(out, "\n")
}

func formatAny(value any) string {
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return strings.TrimSpace(v.(string))
	}
}
