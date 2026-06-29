package configs

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Bot           BotConfig          `mapstructure:"bot"`
	Database      DatabaseConfig     `mapstructure:"database"`
	Log           LogConfig          `mapstructure:"log"`
	UsersJID      UsersJID           `mapstructure:"usersJID"`
	ApiURLs       ApiURLs            `mapstructure:"apiUrls"`
	ScheduledJobs []WeekdayJobConfig `mapstructure:"scheduledJobs"`
	Mikael        MikaelConfig       `mapstructure:"mikael"`

	Groq   GroqConfig
	Tavily TavilyConfig
	Music  MusicConfig
	Owner  OwnerConfig
}

type WeekdayJobConfig struct {
	Name         string   `mapstructure:"name"`
	Day          string   `mapstructure:"day"`
	Enabled      bool     `mapstructure:"enabled"`
	Hour         int      `mapstructure:"hour"`
	Minute       int      `mapstructure:"minute"`
	AudioPath    string   `mapstructure:"audioPath"`
	StickerName  string   `mapstructure:"stickerName"`
	TargetGroups []string `mapstructure:"targetGroups"`
}

type BotConfig struct {
	Name            string `mapstructure:"name"`
	Prefix          string `mapstructure:"prefix"`
	Environment     string `mapstructure:"environment"`
	Timezone        string `mapstructure:"timezone"`
	NLPGroupTrigger bool   `mapstructure:"nlpGroupTrigger"`
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
	Geocoding    string `mapstructure:"geocoding"`
	OpenMeteoGeo string `mapstructure:"openMeteoGeo"`
	Weather      string `mapstructure:"weather"`
	Cotacao      string `mapstructure:"cotacao"`
	Feriado      string `mapstructure:"feriado"`
}

func Load() *Config {
	_ = godotenv.Load()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("")

	// Mapeamento de env vars → chaves viper
	viper.BindEnv("bot.prefix", "COMMAND_PREFIX")
	viper.BindEnv("bot.environment", "ENVIRONMENT")
	viper.BindEnv("log.level", "LOG_LEVEL")
	viper.BindEnv("database.dsn", "DB_DSN")
	viper.BindEnv("usersjid.owner", "OWNER_JID")
	viper.BindEnv("apicalls.groq.url", "GROQ_URL")
	viper.BindEnv("apicalls.groq.apiKey", "GROQ_API_KEY")
	viper.BindEnv("apicalls.tavily.apiKey", "TAVILY_API_KEY")
	viper.BindEnv("apicalls.music.serverUrl", "MUSIC_SERVER_URL")
	viper.BindEnv("apicalls.music.apiToken", "API_AUTH_TOKEN")
	viper.BindEnv("apicalls.owner.number", "OWNER_NUMBER")

	if err := viper.ReadInConfig(); err != nil {
		log.Println("config.yaml não encontrado, usando env/default")
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatal("Erro ao carregar config:", err)
	}

	// Default para timezone
	if cfg.Bot.Timezone == "" {
		cfg.Bot.Timezone = "America/Sao_Paulo"
	}

	// Preenche campos que vieram de env vars mas não tem chave mapstructure no struct
	cfg.Groq = GroqConfig{
		URL:    viper.GetString("apicalls.groq.url"),
		APIKey: viper.GetString("apicalls.groq.apiKey"),
	}
	cfg.Tavily = TavilyConfig{
		APIKey: viper.GetString("apicalls.tavily.apiKey"),
	}
	cfg.Music = MusicConfig{
		ServerURL: viper.GetString("apicalls.music.serverUrl"),
		APIToken:  viper.GetString("apicalls.music.apiToken"),
	}
	cfg.Owner = OwnerConfig{
		Number: viper.GetString("apicalls.owner.number"),
	}

	return cfg
}

// Configurações carregadas de env vars / viper (sem mapstructure no yaml)
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

type MikaelConfig struct {
	LID    string   `mapstructure:"lid"`
	Groups []string `mapstructure:"groups"`
}
