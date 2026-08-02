package logger

type FileConfig struct {
	Name    string `koanf:"name" validate:"required"`
	Size    int64  `koanf:"size" validate:"gt=0"`
	Backups int    `koanf:"backups" validate:"min=0"` // number of old log files to retain. 0 means to retain all old log files.

	ChannelSize uint `koanf:"channel_size" validate:"gt=0"` // size of the channel for async writing
	Discard     bool `koanf:"discard"`                      // whether to discard log entries when the channel is full. If false, log entries will be blocked until there is space in the channel.
}

type Config struct {
	Level  string     `koanf:"level" validate:"required,oneof=trace debug info warn error fatal panic"`
	Caller int        `koanf:"caller" validate:"min=0"` // whether to add source code information (file and line number) to log entries. 0 disables this feature, and any positive integer enables it with the specified call depth.
	File   FileConfig `koanf:"file" validate:"required"`
}

var defaultCfg = Config{
	Level:  "info",
	Caller: 0,
	File: FileConfig{
		Size:        134217728, // 128 MiB
		Backups:     7,
		ChannelSize: 4096,
		Discard:     true,
	},
}
