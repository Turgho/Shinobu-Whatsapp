package configs

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Environment string `env:"ENVIRONMENT"`

	ApiURLs  ApiURLs
	UsersJID UsersJID
	Bot      BotConfig
	Database DatabaseConfig
	Log      LogConfig
}

type BotConfig struct {
	Name   string `env:"BOT_NAME"`
	Prefix string `env:"COMMAND_PREFIX"`
}

type DatabaseConfig struct {
	Driver string `env:"DB_DRIVER"`
	Dsn    string `env:"DB_DSN"`
}

type LogConfig struct {
	Level string `env:"LOG_LEVEL"`
}

type UsersJID struct {
	Owner  string   `env:"OWNER_JID"`
	Admins []string `env:"ADMINS_JID"`
}

type ApiURLs struct {
	Geocoding string `env:"GEOCODING_API_URL"`
	Weather   string `env:"WEATHER_API_URL"`
}

func Load() *Config {
	// carrega .env
	_ = godotenv.Load()

	// configura o viper para ler o config.yaml e as variáveis de ambiente
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./internal/configs")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		log.Println("Config YAML não encontrado, usando env/default")
	}

	cfg := &Config{}

	// unmarshal o config para a struct Config
	err = viper.Unmarshal(&cfg)
	if err != nil {
		log.Fatal("Erro ao carregar config:", err)
	}

	return cfg
}
