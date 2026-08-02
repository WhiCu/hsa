package config

import (
	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
)

func Package(path string) func(do.Injector) {
	return do.Package(
		do.Lazy(func(do.Injector) (*koanf.Koanf, error) {
			return NewKoanf(path)
		}),
	)
}
