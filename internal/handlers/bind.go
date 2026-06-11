package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

func formField(r *http.Request, key string) string {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		var data map[string]any
		if err := json.NewDecoder(r.Body).Decode(&data); err == nil {
			if v, ok := data[key].(string); ok {
				return v
			}
		}
		return ""
	}
	_ = r.ParseForm()
	return r.FormValue(key)
}

// Bind auto-detects Content-Type and decodes request body into T.
// Supports: application/json, application/x-www-form-urlencoded, multipart/form-data
func Bind[T any](r *http.Request) (T, error) {
	var dst T
	ct := r.Header.Get("Content-Type")

	switch {
	case strings.Contains(ct, "application/json"):
		if err := json.NewDecoder(r.Body).Decode(&dst); err != nil {
			return dst, fmt.Errorf("json: %w", err)
		}
	case strings.Contains(ct, "application/x-www-form-urlencoded"),
		strings.Contains(ct, "multipart/form-data"):
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			_ = r.ParseForm()
		}
		if err := mapForm(r, &dst); err != nil {
			return dst, fmt.Errorf("form: %w", err)
		}
	default:
		// fallback: try JSON
		if err := json.NewDecoder(r.Body).Decode(&dst); err != nil {
			return dst, fmt.Errorf("unsupported content-type %q", ct)
		}
	}

	return dst, nil
}

// mapForm maps r.Form values into a struct using `form:"key"` tags.
// Falls back to lowercased field name if no tag.
func mapForm(r *http.Request, dst any) error {
	v := reflect.ValueOf(dst).Elem()
	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}

		key := field.Tag.Get("form")
		if key == "" {
			key = field.Tag.Get("json")
		}
		if key == "" {
			key = strings.ToLower(field.Name)
		}
		key, _, _ = strings.Cut(key, ",") // strip ",omitempty" etc

		val := r.FormValue(key)
		if val == "" {
			continue
		}

		switch fv.Kind() {
		case reflect.String:
			fv.SetString(val)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return fmt.Errorf("field %s: %w", key, err)
			}
			fv.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return fmt.Errorf("field %s: %w", key, err)
			}
			fv.SetUint(n)
		case reflect.Float32, reflect.Float64:
			n, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return fmt.Errorf("field %s: %w", key, err)
			}
			fv.SetFloat(n)
		case reflect.Bool:
			b, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("field %s: %w", key, err)
			}
			fv.SetBool(b)
		}
	}
	return nil
}
