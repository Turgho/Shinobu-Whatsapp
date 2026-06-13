package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Environment string `mapstructure:"environment"`

	ApiURLs  ApiURLs  `mapstructure:"apiUrls"`
	UsersJID UsersJID `mapstructure:"usersJID"`
	Bot      BotConfig `mapstructure:"bot"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig `mapstructure:"log"`

	Groq   GroqConfig
	Tavily TavilyConfig
	Music  MusicConfig
	Owner  OwnerConfig
	Football FootballConfig `mapstructure:"football"`
}

type BotConfig struct {
	Name   string `mapstructure:"name"`
	Prefix string `mapstructure:"prefix"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	Dsn    string `mapstructure:"dsn"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type UsersJID struct {
	Owner  string   `mapstructure:"owner"`
	Admins []string `mapstructure:"admins"`
}

type ApiURLs struct {
	Geocoding string `mapstructure:"geocoding"`
	Weather   string `mapstructure:"weather"`
}

type GroqConfig struct {
	URL    string
	APIKey string
}

type TavilyConfig struct {
	APIKey string
}

type MusicConfig struct {
	ServerURL string
	APIToken  string
}

type OwnerConfig struct {
	Number string
}

type FootballTeam struct {
	Name      string `mapstructure:"name"`
	APITeamID int    `mapstructure:"api_team_id"`
	Flag      string `mapstructure:"flag"`
}

type FootballConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	APIKey      string   `mapstructure:"api_key"`
	NotifyJID   string   `mapstructure:"notify_jid"`
	PollInterval PollIntervalConfig `mapstructure:"poll"`
	WatchedTeams []FootballTeam `mapstructure:"watched_teams"`
}

type PollIntervalConfig struct {
	IdleInterval string `mapstructure:"idle_interval"` // e.g., "5m"
	LiveInterval string `mapstructure:"live_interval"` // e.g., "15s"
}

func Load() *Config {
	_ = godotenv.Load()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Println("config.yaml não encontrado, usando env/default")
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatal("Erro ao carregar config:", err)
	}

	cfg.Groq = GroqConfig{
		URL:    os.Getenv("GROQ_URL"),
		APIKey: os.Getenv("GROQ_API_KEY"),
	}
	cfg.Tavily = TavilyConfig{
		APIKey: os.Getenv("TAVILY_API_KEY"),
	}
	cfg.Music = MusicConfig{
		ServerURL: os.Getenv("MUSIC_SERVER_URL"),
		APIToken:  os.Getenv("API_AUTH_TOKEN"),
	}
	cfg.Owner = OwnerConfig{
		Number: os.Getenv("OWNER_NUMBER"),
	}
	// Football
	if apiKey := os.Getenv("FOOTBALL_API_KEY"); apiKey != "" {
		cfg.Football.APIKey = apiKey
	}

	return cfg
}
