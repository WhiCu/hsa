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
			f, err := os.CreateTemp(GinkgoT().TempDir(), "config-test-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			tempFile = f.Name()
			err = f.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should test env provider transformer logic", func() {
			err := os.WriteFile(tempFile, []byte("key: value"), 0644)
			Expect(err).NotTo(HaveOccurred())

			GinkgoT().Setenv("APP_FOO_BAR", "test")
			GinkgoT().Setenv("APP_ARRAY_VAL", "one two three")

			fsys, name, err := config.ResolveDiskFS(tempFile)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(fsys, name)
			Expect(err).NotTo(HaveOccurred())
			Expect(k).NotTo(BeNil())

			Expect(k.String("foo.bar")).To(Equal("test"))
			Expect(k.Strings("array.val")).To(Equal([]string{"one", "two", "three"}))

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
			f, err := os.CreateTemp(GinkgoT().TempDir(), "config-test-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			tempFile = f.Name()
			err = f.Close()
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
			f, err := os.CreateTemp(GinkgoT().TempDir(), "config-test-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			tempFile = f.Name()
			err = f.Close()
			Expect(err).NotTo(HaveOccurred())
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
hmac_val: my_hmac
prf_info: my_prf
seed_data: my_seed
a_nonce: the_nonce_value
metadata_blob: a_metadata
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
			Expect(dump).To(ContainSubstring("hmac_val = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("prf_info = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("seed_data = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("a_nonce = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("metadata_blob = ***REDACTED***"))

			Expect(dump).NotTo(ContainSubstring("hidden_value"))
			Expect(dump).NotTo(ContainSubstring("password123"))
			Expect(dump).NotTo(ContainSubstring("some_api_key"))
			Expect(dump).NotTo(ContainSubstring("bearer_token_abc"))
			Expect(dump).NotTo(ContainSubstring("nested_password"))
			Expect(dump).NotTo(ContainSubstring("my_hmac"))
			Expect(dump).NotTo(ContainSubstring("my_prf"))
			Expect(dump).NotTo(ContainSubstring("my_seed"))
			Expect(dump).NotTo(ContainSubstring("the_nonce_value"))
			Expect(dump).NotTo(ContainSubstring("a_metadata"))
		})
	})

	Describe("ResolveDiskFS", func() {
		It("should correctly split absolute path into FS and filename", func() {
			f, err := os.CreateTemp(GinkgoT().TempDir(), "config-test-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			tempFile := f.Name()
			err = f.Close()
			Expect(err).NotTo(HaveOccurred())

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
			f, err := os.CreateTemp(GinkgoT().TempDir(), "config-test-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			tempFile = f.Name()
			err = f.Close()
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(tempFile, []byte("pkg_key: pkg_value"), 0644)
			Expect(err).NotTo(HaveOccurred())
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
