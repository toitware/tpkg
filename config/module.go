package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var Module = fx.Provide(
	loadConfig,
	provideConfig,
)

type Config struct {
	Port      int    `mapstructure:"port"`
	DebugPort *int   `mapstructure:"debug_port"`
	WebPath   string `mapstructure:"web_path"`
	HTTPS     bool   `mapstructure:"https"`

	Registry Registry `mapstructure:"registry"`

	Logging  Logging  `mapstructure:"logging"`
	Metrics  Metrics  `mapstructure:"metrics"`
	Toitdocs Toitdocs `mapstructure:"toitdocs"`
}

type Logging struct {
	Backend string            `mapstructure:"backend"`
	Level   string            `mapstructure:"level"`
	Tags    map[string]string `mapstructure:"tags"`
}

type Metrics struct {
	Enabled bool              `mapstructure:"enabled"`
	Tags    map[string]string `mapstructure:"tags"`
	Prefix  string            `mapstructure:"prefix"`
}

type Registry struct {
	Url        string `mapstructure:"url"`
	Branch     string `mapstructure:"branch"`
	CachePath  string `mapstructure:"cache_path"`
	SSHKeyFile string `mapstructure:"ssh_key_file_file"`
}

type SDK struct {
	Path         string `mapstructure:"path"`
	ToitcPath_   string `mapstructure:"toitc_path"`
	ToitlspPath_ string `mapstructure:"toitlsp_path"`
}

func (s *SDK) ToitcPath() string {
	if s.ToitcPath_ == "" {
		return filepath.Join(s.Path, "toitc")
	}
	return s.ToitcPath_
}

func (s *SDK) ToitlspPath() string {
	if s.ToitlspPath_ == "" {
		return filepath.Join(s.Path, "toitlsp")
	}
	return s.ToitlspPath_
}

type Toitdocs struct {
	CachePath  string `mapstructure:"cache_path"`
	ViewerPath string `mapstructure:"viewer_path"`
	SDK        SDK    `mapstructure:"sdk"`
}

func provideConfig(cfg *viper.Viper) (*Config, error) {
	res := &Config{}
	if err := cfg.Unmarshal(res); err != nil {
		return nil, err
	}

	return res, nil
}

func loadConfig(log fx.Printer) (*viper.Viper, error) {
	res := viper.New()

	if cfgPath, ok := os.LookupEnv("CONFIG_PATH"); ok {
		res.AddConfigPath(cfgPath)
	}
	res.AddConfigPath("./config")

	res.SetConfigName("config")

	log.Printf("loading config file %s.yaml\n", "base")
	if err := res.ReadInConfig(); err != nil {
		return nil, err
	}

	envSubstitution(res)

	return res, nil
}

func envSubstitution(v *viper.Viper) {
	for _, k := range v.AllKeys() {
		v.Set(k, recEnvSubstitution(v.Get(k)))
	}
}

func recEnvSubstitution(in interface{}) interface{} {
	switch val := in.(type) {
	case string:
		return ExpandEnv(val)
	case []string:
		for i, v := range val {
			val[i] = ExpandEnv(v)
		}
		return val
	case map[string]string:
		for k, v := range val {
			val[k] = ExpandEnv(v)
		}
		return val
	case []interface{}:
		for i, v := range val {
			val[i] = recEnvSubstitution(v)
		}
		return val
	case map[interface{}]interface{}:
		for k, v := range val {
			val[k] = recEnvSubstitution(v)
		}
		return val
	case map[string]interface{}:
		for k, v := range val {
			val[k] = recEnvSubstitution(v)
		}
		return val
	default:
		return in
	}
}

func ExpandWithDefault(val string, mapping func(env string) (string, bool)) string {
	return os.Expand(val, func(env string) string {
		envSet := strings.SplitN(env, ":", 2)
		if len(envSet) > 0 {
			env = envSet[0]
			def := ""
			if len(envSet) == 2 {
				def = envSet[1]
			}
			if val, ok := mapping(env); ok {
				return val
			}
			return def
		}
		return val
	})
}

func ExpandEnv(val string) string {
	return ExpandWithDefault(val, os.LookupEnv)
}
