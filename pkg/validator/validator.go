package validator

import (
	"fmt"
	"sort"
	"strings"
)

type FieldError struct {
	Field string
	Err   error
}

func (e FieldError) Error() string {
	if e.Err == nil {
		return e.Field + ": validation failed"
	}
	return e.Field + ": " + e.Err.Error()
}

type ValidationError struct {
	Errors []FieldError
}

func (e ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	parts := make([]string, 0, len(e.Errors))
	for i := range e.Errors {
		parts = append(parts, e.Errors[i].Error())
	}
	return strings.Join(parts, "; ")
}

func (e ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

type Validator struct {
	rules map[string][]Rule
}

func New() *Validator {
	return &Validator{rules: make(map[string][]Rule)}
}

func (v *Validator) Add(field string, rules ...Rule) *Validator {
	if v.rules == nil {
		v.rules = make(map[string][]Rule)
	}
	v.rules[field] = append(v.rules[field], rules...)
	return v
}

func (v *Validator) Validate(values map[string]any) error {
	if v == nil || len(v.rules) == 0 {
		return nil
	}
	result := ValidationError{Errors: make([]FieldError, 0)}
	fields := make([]string, 0, len(v.rules))
	for field := range v.rules {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	for _, field := range fields {
		value := values[field]
		for _, rule := range v.rules[field] {
			if rule == nil {
				continue
			}
			if err := rule(value); err != nil {
				result.Errors = append(result.Errors, FieldError{Field: field, Err: err})
				break
			}
		}
	}

	if !result.HasErrors() {
		return nil
	}
	return result
}

func Validate(values map[string]any, defs map[string][]Rule) error {
	v := New()
	for field, rules := range defs {
		v.Add(field, rules...)
	}
	if err := v.Validate(values); err != nil {
		return err
	}
	return nil
}

func MustValidate(values map[string]any, defs map[string][]Rule) {
	if err := Validate(values, defs); err != nil {
		panic(fmt.Sprintf("validator: %v", err))
	}
}
