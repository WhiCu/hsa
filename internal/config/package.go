package config

import (
	"io/fs"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
)

func Package(fs fs.FS, path string) func(do.Injector) {
	return do.Package(
		do.Lazy(func(do.Injector) (*koanf.Koanf, error) {
			return NewKoanf(fs, path)
		}),
	)
}
