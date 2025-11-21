// config/types.go
package config

type EndpointConfig struct {
	Enabled bool              `yaml:"enabled"`
	Method  string            `yaml:"method"`
	Paths   map[string]string `yaml:"paths,omitempty"`
	Path    string            `yaml:"path,omitempty"`
}

// Token endpoint authentication
type TokenEndpoint struct {
	URL              string            `yaml:"url"`
	Method           string            `yaml:"method"`
	AuthorizationKey string            `yaml:"authorizationKey,omitempty"` // Bearer
	Username         string            `yaml:"username,omitempty"`         // Basic Auth or form
	Password         string            `yaml:"password,omitempty"`         // Basic Auth or form
	FormUsername     string            `yaml:"form_username,omitempty"`    // form field
	FormPassword     string            `yaml:"form_password,omitempty"`    // form field
	ExtraParams      map[string]string `yaml:"extra_params,omitempty"`
}

type CallbackConfig struct {
	Enabled        bool           `yaml:"enabled"`
	URL            string         `yaml:"url"`
	TimeoutSeconds int            `yaml:"timeout_seconds"`
	RetryCount     int            `yaml:"retry_count"`
	TokenEndpoint  *TokenEndpoint `yaml:"token_endpoint,omitempty"`
}

type Config struct {
	Redis struct {
		URL string `yaml:"url"`
	} `yaml:"redis"`

	Upstream map[string]string `yaml:"upstream"`

	AttendanceCallback CallbackConfig `yaml:"attendance_callback"`

	Endpoints struct {
		Data          EndpointConfig `yaml:"data"`
		Attendance    EndpointConfig `yaml:"attendance"`
		Reset         EndpointConfig `yaml:"reset"`
		MappingUpload EndpointConfig `yaml:"mapping_upload"`
	} `yaml:"endpoints"`
}

