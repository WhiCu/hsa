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
	Describe("resolvePath", func() {
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
			os.Unsetenv(config.ConfigPath)
		})

		It("should use PATH_CONFIG environment variable if set", func() {
			os.Setenv(config.ConfigPath, tempFile)

			// resolvePath is internal, so we need to test it through NewKoanf or export it for tests.
			// However, since it's unexported, we'll test it via NewKoanf, which fails or succeeds based on it.
			k, err := config.NewKoanf("non-existent-default.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(k).NotTo(BeNil())
		})

		It("should test env provider transformer logic", func() {
			err := os.WriteFile(tempFile, []byte("key: value"), 0644)
			Expect(err).NotTo(HaveOccurred())

			os.Setenv("APP_FOO_BAR", "test")
			os.Setenv("APP_ARRAY_VAL", "one two three")

			k, err := config.NewKoanf(tempFile)
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

			k, err := config.NewKoanf(tempFile)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("load yaml config"))
			Expect(k).To(BeNil())
		})

		It("should fallback to default path if PATH_CONFIG is not set", func() {
			k, err := config.NewKoanf(tempFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(k).NotTo(BeNil())
		})

		It("should return error if path is empty", func() {
			k, err := config.NewKoanf("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resolve config path"))
			Expect(err.Error()).To(ContainSubstring(config.ErrPathNotSet.Error()))
			Expect(k).To(BeNil())
		})

		It("should return error if file does not exist", func() {
			k, err := config.NewKoanf("non-existent-default.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resolve config path"))
			Expect(err.Error()).To(ContainSubstring(config.ErrPathNotExist.Error()))
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
			f.Close()
		})

		AfterEach(func() {
			os.Remove(tempFile)
		})

		It("should successfully get and validate config", func() {
			err := os.WriteFile(tempFile, []byte("test:\n  name: 'test-name'\n  value: 42"), 0644)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(tempFile)
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

			k, err := config.NewKoanf(tempFile)
			Expect(err).NotTo(HaveOccurred())

			var def TestConfig
			_, err = config.GetConfig(k, "test", &def)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validate config for key"))
		})

		It("should return error if unmarshal fails", func() {
			err := os.WriteFile(tempFile, []byte("test:\n  name: []"), 0644)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(tempFile)
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

		It("should dump flat config correctly", func() {
			err := os.WriteFile(tempFile, []byte("some_field: value"), 0644)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(tempFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(k).NotTo(BeNil())

			dump := config.DumpFlat(k)
			Expect(dump).To(ContainSubstring("some_field = value"))
		})

		It("should redact sensitive keys in dump flat config", func() {
			err := os.WriteFile(tempFile, []byte("secret_field: secret123\nnormal_key: pass_123\ntoken_field: abc\npassword: 123"), 0644)
			Expect(err).NotTo(HaveOccurred())

			k, err := config.NewKoanf(tempFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(k).NotTo(BeNil())

			dump := config.DumpFlat(k)
			Expect(dump).To(ContainSubstring("secret_field = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("normal_key = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("token_field = ***REDACTED***"))
			Expect(dump).To(ContainSubstring("password = ***REDACTED***"))
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
		})

		AfterEach(func() {
			os.Remove(tempFile)
		})

		It("should return a valid di package and successfully initialize NewKoanf", func() {
			err := os.WriteFile(tempFile, []byte("key: value"), 0644)
			Expect(err).NotTo(HaveOccurred())

			pkg := config.Package(tempFile)
			Expect(pkg).NotTo(BeNil())

			i := do.New()
			pkg(i)
			k, err := do.Invoke[*koanf.Koanf](i)
			Expect(err).NotTo(HaveOccurred())
			Expect(k).NotTo(BeNil())
		})
	})
})
