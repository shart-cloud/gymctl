package checks

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

func compareValue(actual string, operator string, expected interface{}, valueType string) (bool, string) {
	if operator == "" {
		operator = "equals"
	}
	if operator == "exists" {
		if actual != "" {
			return true, ""
		}
		return false, "value not found"
	}

	stringExpected := ""
	if expected != nil {
		stringExpected = fmt.Sprintf("%v", expected)
	}

	switch operator {
	case "equals":
		if actual == stringExpected {
			return true, ""
		}
		return false, fmt.Sprintf("expected %s, got %s", stringExpected, actual)
	case "notEquals":
		if actual != stringExpected {
			return true, ""
		}
		return false, fmt.Sprintf("expected not %s, got %s", stringExpected, actual)
	case "contains":
		if strings.Contains(actual, stringExpected) {
			return true, ""
		}
		return false, fmt.Sprintf("expected contains %s, got %s", stringExpected, actual)
	case "regex":
		matched, err := regexp.MatchString(stringExpected, actual)
		if err != nil {
			return false, fmt.Sprintf("invalid regex: %s", err)
		}
		if matched {
			return true, ""
		}
		return false, fmt.Sprintf("expected %s to match %s", actual, stringExpected)
	case "greaterThan", "lessThan":
		return compareOrdered(actual, stringExpected, operator, valueType)
	default:
		return false, fmt.Sprintf("unsupported operator: %s", operator)
	}
}

func compareOrdered(actual string, expected string, operator string, valueType string) (bool, string) {
	switch valueType {
	case "number":
		actualNum, err := strconv.ParseFloat(actual, 64)
		if err != nil {
			return false, fmt.Sprintf("invalid number: %s", actual)
		}
		expectedNum, err := strconv.ParseFloat(expected, 64)
		if err != nil {
			return false, fmt.Sprintf("invalid expected number: %s", expected)
		}
		if operator == "greaterThan" {
			if actualNum > expectedNum {
				return true, ""
			}
			return false, fmt.Sprintf("expected %v > %v", actualNum, expectedNum)
		}
		if actualNum < expectedNum {
			return true, ""
		}
		return false, fmt.Sprintf("expected %v < %v", actualNum, expectedNum)
	case "quantity":
		actualQty, err := resource.ParseQuantity(actual)
		if err != nil {
			return false, fmt.Sprintf("invalid quantity: %s", actual)
		}
		expectedQty, err := resource.ParseQuantity(expected)
		if err != nil {
			return false, fmt.Sprintf("invalid expected quantity: %s", expected)
		}
		cmp := actualQty.Cmp(expectedQty)
		if operator == "greaterThan" {
			if cmp > 0 {
				return true, ""
			}
			return false, fmt.Sprintf("expected %s > %s", actualQty.String(), expectedQty.String())
		}
		if cmp < 0 {
			return true, ""
		}
		return false, fmt.Sprintf("expected %s < %s", actualQty.String(), expectedQty.String())
	default:
		return false, "valueType required for ordered comparison"
	}
}

func compareInt(actual int64, expected int64, operator string) (bool, string) {
	if operator == "" {
		operator = "equals"
	}
	switch operator {
	case "equals":
		if actual == expected {
			return true, ""
		}
		return false, fmt.Sprintf("expected %d, got %d", expected, actual)
	case "notEquals":
		if actual != expected {
			return true, ""
		}
		return false, fmt.Sprintf("expected not %d, got %d", expected, actual)
	case "greaterThan":
		if actual > expected {
			return true, ""
		}
		return false, fmt.Sprintf("expected %d > %d", actual, expected)
	case "lessThan":
		if actual < expected {
			return true, ""
		}
		return false, fmt.Sprintf("expected %d < %d", actual, expected)
	default:
		return false, fmt.Sprintf("unsupported operator: %s", operator)
	}
}

func parseSize(value string) (int64, error) {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return 0, fmt.Errorf("size is empty")
	}

	multiplier := int64(1)
	switch {
	case strings.HasSuffix(value, "GB"):
		multiplier = 1024 * 1024 * 1024
		value = strings.TrimSuffix(value, "GB")
	case strings.HasSuffix(value, "MB"):
		multiplier = 1024 * 1024
		value = strings.TrimSuffix(value, "MB")
	case strings.HasSuffix(value, "KB"):
		multiplier = 1024
		value = strings.TrimSuffix(value, "KB")
	case strings.HasSuffix(value, "B"):
		multiplier = 1
		value = strings.TrimSuffix(value, "B")
	}

	numeric, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %s", value)
	}
	return int64(numeric * float64(multiplier)), nil
}
