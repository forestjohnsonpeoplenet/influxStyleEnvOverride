package influxStyleEnvOverride

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type keyValueRetriever interface {
	get(key string) string
}

type environmentVariableKeyValueRetriever struct{}

func (this environmentVariableKeyValueRetriever) get(key string) string {
	return os.Getenv(key)
}

// ApplyEnvOverrides apply any convention-driven environment varibles on top of the original object.
func ApplyInfluxStyleEnvOverrides(prefix string, originalObject *interface{}) error {
	return applyEnvOverrides(environmentVariableKeyValueRetriever{}, prefix, reflect.ValueOf(originalObject))
}

func applyEnvOverrides(kv keyValueRetriever, prefix string, spec reflect.Value) error {

	// If we have a pointer, dereference it
	s := spec
	if spec.Kind() == reflect.Ptr {
		s = spec.Elem()
	}

	// Make sure we have struct
	if s.Kind() != reflect.Struct {
		return nil
	}

	typeOfSpec := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		// Get the toml tag to determine what env var name to use
		configName := typeOfSpec.Field(i).Tag.Get("toml")
		if configName == "" {
			configName = typeOfSpec.Field(i).Tag.Get("json")
		}
		if configName == "" {
			configName = typeOfSpec.Field(i).Name
		}
		// Replace hyphens with underscores to avoid issues with shells
		configName = strings.Replace(configName, "-", "_", -1)
		fieldKey := typeOfSpec.Field(i).Name

		// Use the upper-case prefix and toml name for the env var
		key := strings.ToUpper(configName)
		if prefix != "" {
			key = strings.ToUpper(fmt.Sprintf("%s_%s", prefix, configName))
		}

		//fmt.Printf("%v, %v, %v\n", key, f.Kind(), f.CanSet())

		// If it's a sub-config, recursively apply
		if f.Kind() == reflect.Struct || f.Kind() == reflect.Ptr {
			if err := applyEnvOverrides(kv, key, f); err != nil {
				return err
			}
			continue
		}

		// Skip any fields that we cannot set
		canSet := f.CanSet() || f.Kind() == reflect.Slice

		value := kv.get(key)

		if !canSet && value != "" {
			//fmt.Printf("failed to apply %v to %v: %v is not settable according to golang reflection", key, fieldKey, fieldKey)
			return fmt.Errorf("failed to apply %v to %v: %v is not settable according to golang reflection", key, fieldKey, fieldKey)
		}
		if canSet {
			//fmt.Printf("%v=%v\n", key, value)
		}

		if canSet {
			// If the type is s slice, apply to each using the index as a suffix
			// e.g. GRAPHITE_0
			if f.Kind() == reflect.Slice || f.Kind() == reflect.Array {
				for i := 0; i < f.Len(); i++ {
					if err := applyEnvOverrides(kv, key, f.Index(i)); err != nil {
						return err
					}
					if err := applyEnvOverrides(kv, fmt.Sprintf("%s_%d", key, i), f.Index(i)); err != nil {
						return err
					}
				}
				continue
			}

			// Skip any fields we don't have a value to set
			if value == "" {
				continue
			}

			switch f.Kind() {
			case reflect.String:
				f.SetString(value)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

				var intValue int64

				// Handle toml.Duration
				if f.Type().Name() == "Duration" {
					dur, err := time.ParseDuration(value)
					if err != nil {
						return fmt.Errorf("failed to apply %v to %v using type %v and value '%v'", key, fieldKey, f.Type().String(), value)
					}
					intValue = dur.Nanoseconds()
				} else {
					var err error
					intValue, err = strconv.ParseInt(value, 0, f.Type().Bits())
					if err != nil {
						return fmt.Errorf("failed to apply %v to %v using type %v and value '%v'", key, fieldKey, f.Type().String(), value)
					}
				}

				f.SetInt(intValue)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				var intValue uint64
				var err error
				intValue, err = strconv.ParseUint(value, 0, f.Type().Bits())
				if err != nil {
					return fmt.Errorf("failed to apply %v to %v using type %v and value '%v'", key, fieldKey, f.Type().String(), value)
				}

				f.SetUint(intValue)
			case reflect.Bool:
				boolValue, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("failed to apply %v to %v using type %v and value '%v'", key, fieldKey, f.Type().String(), value)

				}
				f.SetBool(boolValue)
			case reflect.Float32, reflect.Float64:
				floatValue, err := strconv.ParseFloat(value, f.Type().Bits())
				if err != nil {
					return fmt.Errorf("failed to apply %v to %v using type %v and value '%v'", key, fieldKey, f.Type().String(), value)

				}
				f.SetFloat(floatValue)
			default:
				if err := applyEnvOverrides(kv, key, f); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
