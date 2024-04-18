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
	// Login: user and password
	Username                       string `yaml:"username" validate:"required"`
	Password                       string `yaml:"password" validate:"required"`
	SessionCookieAuthenticationKey string `yaml:"session_cookie_authentication_key" validate:"required"`
	SessionStoreEncryptionKey      string `yaml:"session_store_encryption_key" validate:"required"`
	Services                       struct {
		APIGW struct {
			Addr string `yaml:"addr"`
		} `yaml:"apigw"`
		MockAS struct {
			Addr string `yaml:"addr"`
		} `yaml:"mockas"`
	} `yaml:"services"`
}

// Cfg is the main configuration structure for this application
type Cfg struct {
	Common Common `yaml:"common"`
	Web1   Web1   `yaml:"web1"`
}
