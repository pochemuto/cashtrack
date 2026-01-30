package cashtrack

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

func numericFromString(value string) (pgtype.Numeric, error) {
	var numeric pgtype.Numeric
	if strings.TrimSpace(value) == "" {
		return numeric, fmt.Errorf("amount is empty")
	}
	if err := numeric.Scan(value); err != nil {
		return numeric, err
	}
	return numeric, nil
}

func numericToString(value pgtype.Numeric) string {
	if !value.Valid {
		return ""
	}
	plan := (pgtype.NumericCodec{}).PlanEncode(nil, 0, pgtype.TextFormatCode, value)
	if plan == nil {
		return ""
	}
	buf, err := plan.Encode(value, nil)
	if err != nil {
		return ""
	}
	return string(buf)
}

func numericToFloat(value pgtype.Numeric) (float64, error) {
	if !value.Valid {
		return 0, nil
	}
	raw := numericToString(value)
	if raw == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func numericToCents(value pgtype.Numeric) (int64, error) {
	if !value.Valid {
		return 0, nil
	}
	raw := strings.TrimSpace(numericToString(value))
	if raw == "" {
		return 0, nil
	}
	sign := int64(1)
	if strings.HasPrefix(raw, "-") {
		sign = -1
		raw = strings.TrimPrefix(raw, "-")
	} else if strings.HasPrefix(raw, "+") {
		raw = strings.TrimPrefix(raw, "+")
	}
	parts := strings.SplitN(raw, ".", 2)
	wholePart := parts[0]
	fractionPart := ""
	if len(parts) > 1 {
		fractionPart = parts[1]
	}
	if wholePart == "" {
		wholePart = "0"
	}
	whole, err := strconv.ParseInt(wholePart, 10, 64)
	if err != nil {
		return 0, err
	}
	if len(fractionPart) > 2 {
		fractionPart = fractionPart[:2]
	}
	for len(fractionPart) < 2 {
		fractionPart += "0"
	}
	fraction := int64(0)
	if fractionPart != "" {
		parsed, err := strconv.ParseInt(fractionPart, 10, 64)
		if err != nil {
			return 0, err
		}
		fraction = parsed
	}
	return sign * (whole*100 + fraction), nil
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func centsFromFloat(value float64) int64 {
	return int64(math.Round(value * 100))
}
