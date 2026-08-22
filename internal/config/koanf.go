package config

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const ConfigPath = "PATH_CONFIG"

var Validate = validator.New(validator.WithRequiredStructEnabled())

func NewKoanf(path string) (*koanf.Koanf, error) {
	path, err := resolvePath(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	var conf = koanf.Conf{
		Delim:       ".",
		StrictMerge: true,
	}
	var k = koanf.NewWithConf(conf)

	err = k.Load(file.Provider(path), yaml.Parser())
	if err != nil {
		return nil, fmt.Errorf("load yaml config: %w", err)
	}
	err = k.Load(env.Provider(".", env.Opt{
		Prefix: "APP_",
		TransformFunc: func(key, value string) (string, any) {
			key = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(key, "APP_")), "_", ".")

			if strings.Contains(value, " ") {
				return key, strings.Split(value, " ")
			}

			return key, value
		},
	}), nil)

	// uncovered: load env provider never returns an error natively
	if err != nil {
		return nil, fmt.Errorf("load env config: %w", err)
	}

	return k, nil
}

func GetConfig[T any](k *koanf.Koanf, key string, def *T) (T, error) {
	if err := k.Unmarshal(key, def); err != nil {
		return *def, fmt.Errorf("unmarshal config for key '%s': %w", key, err)
	}
	if err := Validate.Struct(def); err != nil {
		return *def, fmt.Errorf("validate config for key '%s': %w", key, err)
	}
	return *def, nil
}

func resolvePath(def string) (string, error) {
	path := cmp.Or(os.Getenv(ConfigPath), def)
	if path == "" {
		return "", ErrPathNotSet
	}
	cleanPath := filepath.Clean(path)
	if _, err := os.Stat(cleanPath); errors.Is(err, os.ErrNotExist) {
		return "", ErrPathNotExist
	}

	return path, nil
}

func DumpFlat(k *koanf.Koanf) string {
	var sb strings.Builder
	for _, key := range k.Keys() {
		value := k.Get(key)
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "key") || strings.Contains(lowerKey, "pass") || strings.Contains(lowerKey, "token") {
			fmt.Fprintf(&sb, "%s = ***REDACTED***\n", key)
		} else {
			fmt.Fprintf(&sb, "%s = %v\n", key, value)
		}
	}
	return sb.String()
}
