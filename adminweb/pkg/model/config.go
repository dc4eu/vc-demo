package model

// Log holds the log configuration
type Log struct {
	Level      string `yaml:"level"`
	FolderPath string `yaml:"folder_path"`
}

// Common holds the common configuration
type Common struct {
	HTTPProxy  string `yaml:"http_proxy"`
	Production bool   `yaml:"production"`
	Log        Log    `yaml:"log"`
}

type Web1 struct {
	Username string `yaml:"username" validate:"required"`
	Password string `yaml:"password" validate:"required"`
}

// Cfg is the main configuration structure for this application
type Cfg struct {
	Common Common `yaml:"common"`
	Web1   Web1   `yaml:"web1"`
}
