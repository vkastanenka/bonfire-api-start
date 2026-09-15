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

// Normalize recursively applies string sanitization rules using cached type metadata.
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

func getOrBuildPlan(typ reflect.Type) *structPlan {
	if cached, ok := planCache.Load(typ); ok {
		return cached.(*structPlan)
	}

	plan := &structPlan{}
	buildPlan(typ, nil, plan)

	actual, _ := planCache.LoadOrStore(typ, plan)
	return actual.(*structPlan)
}

func buildPlan(typ reflect.Type, indexPrefix []int, plan *structPlan) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		indexPath := make([]int, len(indexPrefix)+1)
		copy(indexPath, indexPrefix)
		indexPath[len(indexPrefix)] = i

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		kind := field.Type.Kind()

		// Recurse into nested structs
		if kind == reflect.Struct {
			buildPlan(field.Type, indexPath, plan)
			continue
		}

		// Recurse into nested struct pointers (*MyStruct)
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

func executePlan(val reflect.Value, plan *structPlan) {
	for _, f := range plan.fields {
		curr := val
		var nilEncountered bool

		// Walk through parent struct hierarchy indices
		for i, idx := range f.index {
			if curr.Kind() == reflect.Ptr {
				if curr.IsNil() {
					nilEncountered = true
					break
				}
				curr = curr.Elem()
			}

			curr = curr.Field(idx)

			// Dereference intermediate struct pointers along the path (not the leaf field itself)
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

		// Handle target leaf field (*string vs string)
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
