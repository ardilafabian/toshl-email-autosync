package common

import "regexp"

func ExtractFieldsStringWithRegexp(s string, r *regexp.Regexp) map[string]string {
	match := r.FindStringSubmatch(s)
	result := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 && name != "" && i < len(match) {
			result[name] = match[i]
		}
	}

	return result
}

func ContainsAllRequiredFields(fields map[string]string) bool {
	requiredFields := []string{"value", "type", "place", "account"}
	requiredFieldsSet := map[string]struct{}{}
	for _, field := range requiredFields {
		requiredFieldsSet[field] = struct{}{}
	}

	for field := range requiredFieldsSet {
		_, ok := fields[field]
		if !ok {
			return false
		}
	}

	return true
}
