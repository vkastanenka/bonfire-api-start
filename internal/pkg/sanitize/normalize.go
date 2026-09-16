package sanitize

import (
	"reflect"
	"strings"
	"sync"
)

type transformFunc func(string) string

type fieldPlan struct {
	index      []int
	isPtr      bool
	transforms []transformFunc
}

type structPlan struct {
	fields []fieldPlan
}

var planCache sync.Map // map[reflect.Type]*structPlan

// Normalize recursively sanitizes struct string fields based on `mod` tags using cached metadata.
func Normalize(s any) {
	val := reflect.ValueOf(s)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return
	}

	plan := getOrBuildPlan(elem.Type())
	executePlan(elem, plan)
}

// getOrBuildPlan retrieves a cached struct layout plan or builds and caches a new one.
func getOrBuildPlan(typ reflect.Type) *structPlan {
	if cached, ok := planCache.Load(typ); ok {
		return cached.(*structPlan)
	}

	plan := &structPlan{}
	buildPlan(typ, nil, plan)

	actual, _ := planCache.LoadOrStore(typ, plan)
	return actual.(*structPlan)
}

// buildPlan recursively inspects a struct type to map string fields and their sanitization tags.
func buildPlan(typ reflect.Type, indexPrefix []int, plan *structPlan) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		indexPath := make([]int, len(indexPrefix)+1)
		copy(indexPath, indexPrefix)
		indexPath[len(indexPrefix)] = i

		if field.PkgPath != "" { // Skip unexported fields
			continue
		}

		kind := field.Type.Kind()

		if kind == reflect.Struct {
			buildPlan(field.Type, indexPath, plan)
			continue
		}

		if kind == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
			buildPlan(field.Type.Elem(), indexPath, plan)
			continue
		}

		tag := field.Tag.Get("mod")
		if tag == "" {
			continue
		}

		var isPtr bool
		if kind == reflect.Ptr && field.Type.Elem().Kind() == reflect.String {
			isPtr = true
		} else if kind != reflect.String {
			continue
		}

		transforms := parseTagTransforms(tag)
		if len(transforms) > 0 {
			plan.fields = append(plan.fields, fieldPlan{
				index:      indexPath,
				isPtr:      isPtr,
				transforms: transforms,
			})
		}
	}
}

// parseTagTransforms converts a comma-separated mod tag into an ordered chain of transform functions.
func parseTagTransforms(tag string) []transformFunc {
	var fnChain []transformFunc

	for tag != "" {
		var dir string
		if idx := strings.IndexByte(tag, ','); idx != -1 {
			dir, tag = tag[:idx], tag[idx+1:]
		} else {
			dir, tag = tag, ""
		}

		switch strings.TrimSpace(dir) {
		case "email":
			fnChain = append(fnChain, Email)
		case "text":
			fnChain = append(fnChain, Text)
		case "url":
			fnChain = append(fnChain, URL)
		}
	}

	return fnChain
}

// executePlan traverses a struct instance using a pre-calculated plan and applies sanitization functions to target fields.
func executePlan(val reflect.Value, plan *structPlan) {
	for _, f := range plan.fields {
		curr := val
		var nilEncountered bool

		for i, idx := range f.index {
			if curr.Kind() == reflect.Ptr {
				if curr.IsNil() {
					nilEncountered = true
					break
				}
				curr = curr.Elem()
			}

			curr = curr.Field(idx)

			if i < len(f.index)-1 && curr.Kind() == reflect.Ptr {
				if curr.IsNil() {
					nilEncountered = true
					break
				}
				curr = curr.Elem()
			}
		}

		if nilEncountered {
			continue
		}

		if f.isPtr {
			if curr.Kind() != reflect.Ptr || curr.IsNil() {
				continue
			}
			curr = curr.Elem()
		}

		if !curr.CanSet() || curr.Kind() != reflect.String {
			continue
		}

		str := curr.String()
		for _, tf := range f.transforms {
			str = tf(str)
		}
		curr.SetString(str)
	}
}
