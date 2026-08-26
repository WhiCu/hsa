package config_test

import (
	"os"

	"github.com/knadh/koanf/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/config"
)

var _ = Describe("Config", func() {

	Describe("NewKoanf", func() {
		var (
			tempFile string
		)

		BeforeEach(func() {
			f, err := os.CreateTemp("", "config-test-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			tempFile = f.Name()
			f.Close()
		})

		AfterEach(func() {
			os.Remove(tempFile)
		})

		It("should test env provider transformer logic", func() {
			err := os.WriteFile(tempFile, []byte("key: value"), 0644)
			Expect(err).NotTo(HaveOccurred())

			os.Setenv("APP_FOO_BAR", "test")
			os.Setenv("APP_ARRAY_VAL", "one two three")

			fsys, name, err := config.ResolveDiskFS(tempFile)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(fsys, name)
			Expect(err).NotTo(HaveOccurred())
			Expect(k).NotTo(BeNil())

			Expect(k.String("foo.bar")).To(Equal("test"))
			Expect(k.Strings("array.val")).To(Equal([]string{"one", "two", "three"}))

			os.Unsetenv("APP_FOO_BAR")
			os.Unsetenv("APP_ARRAY_VAL")
		})

		It("should return error if invalid yaml", func() {
			err := os.WriteFile(tempFile, []byte("invalid: yaml: content: -"), 0644)
			Expect(err).NotTo(HaveOccurred())

			fsys, name, err := config.ResolveDiskFS(tempFile)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(fsys, name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("load yaml config"))
			Expect(k).To(BeNil())
		})
	})

	Describe("GetConfig", func() {
		var (
			tempFile string
		)

		type TestConfig struct {
			Name  string `koanf:"name" validate:"required"`
			Value int    `koanf:"value"`
		}

		BeforeEach(func() {
			f, err := os.CreateTemp("", "config-test-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			tempFile = f.Name()
			err = f.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := os.Remove(tempFile)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should successfully get and validate config", func() {
			err := os.WriteFile(tempFile, []byte("test:\n  name: 'test-name'\n  value: 42"), 0644)
			Expect(err).NotTo(HaveOccurred())

			fsys, name, err := config.ResolveDiskFS(tempFile)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(fsys, name)
			Expect(err).NotTo(HaveOccurred())

			var def TestConfig
			cfg, err := config.GetConfig(k, "test", &def)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Name).To(Equal("test-name"))
			Expect(cfg.Value).To(Equal(42))
		})

		It("should return error if validation fails", func() {
			err := os.WriteFile(tempFile, []byte("test:\n  value: 42"), 0644)
			Expect(err).NotTo(HaveOccurred())

			fsys, name, err := config.ResolveDiskFS(tempFile)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(fsys, name)
			Expect(err).NotTo(HaveOccurred())

			var def TestConfig
			_, err = config.GetConfig(k, "test", &def)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validate config for key"))
		})

		It("should return error if unmarshal fails", func() {
			err := os.WriteFile(tempFile, []byte("test:\n  name: []"), 0644)
			Expect(err).NotTo(HaveOccurred())

			fsys, name, err := config.ResolveDiskFS(tempFile)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(fsys, name)
			Expect(err).NotTo(HaveOccurred())

			var def TestConfig
			_, err = config.GetConfig(k, "test", &def)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unmarshal config for key"))
		})
	})

	Describe("DumpFlat", func() {
		var (
			tempFile string
		)

		BeforeEach(func() {
			f, err := os.CreateTemp("", "config-test-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			tempFile = f.Name()
			f.Close()
		})

		AfterEach(func() {
			os.Remove(tempFile)
		})

		It("should dump flat config correctly and redact sensitive fields", func() {
			yamlContent := `
safe_field: visible_value
my_secret: hidden_value
database_pass: password123
api_key: some_api_key
auth_token: bearer_token_abc
nested:
  pass: nested_password
`
			err := os.WriteFile(tempFile, []byte(yamlContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			fsys, name, err := config.ResolveDiskFS(tempFile)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(fsys, name)
			Expect(err).NotTo(HaveOccurred())
			Expect(k).NotTo(BeNil())

			dump := config.DumpFlat(k)
			Expect(dump).To(ContainSubstring("safe_field = visible_value"))
			Expect(dump).To(ContainSubstring("my_secret = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("database_pass = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("api_key = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("auth_token = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("nested.pass = ***REDACTED***"))

			Expect(dump).NotTo(ContainSubstring("hidden_value"))
			Expect(dump).NotTo(ContainSubstring("password123"))
			Expect(dump).NotTo(ContainSubstring("some_api_key"))
			Expect(dump).NotTo(ContainSubstring("bearer_token_abc"))
			Expect(dump).NotTo(ContainSubstring("nested_password"))
		})
	})

	Describe("ResolveDiskFS", func() {
		It("should correctly split absolute path into FS and filename", func() {
			f, err := os.CreateTemp("", "config-test-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			tempFile := f.Name()
			f.Close()
			defer os.Remove(tempFile)

			fsys, name, err := config.ResolveDiskFS(tempFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(fsys).NotTo(BeNil())
			Expect(name).NotTo(BeEmpty())
		})
	})

	Describe("Package", func() {
		var (
			tempFile string
		)

		BeforeEach(func() {
			f, err := os.CreateTemp("", "config-test-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			tempFile = f.Name()
			f.Close()

			err = os.WriteFile(tempFile, []byte("pkg_key: pkg_value"), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Remove(tempFile)
		})

		It("should return a valid di package and successfully initialize Koanf", func() {
			// 1. Сначала резолвим ФС так же, как это будет делать main.go
			fsys, name, err := config.ResolveDiskFS(tempFile)
			Expect(err).NotTo(HaveOccurred())

			// 2. Инициализируем пакет новой сигнатурой
			pkg := config.Package(fsys, name)
			Expect(pkg).NotTo(BeNil())

			// 3. Проверяем инжектор
			i := do.New()
			pkg(i)
			k, err := do.Invoke[*koanf.Koanf](i)
			Expect(err).NotTo(HaveOccurred())
			Expect(k).NotTo(BeNil())

			// Убеждаемся, что конфиг реально загрузился
			Expect(k.String("pkg_key")).To(Equal("pkg_value"))
		})
	})
})
