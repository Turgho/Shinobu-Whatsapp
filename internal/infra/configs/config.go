package configs

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/identity"
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

	Groq     GroqConfig
	Tavily   TavilyConfig
	IA       IAConfig
	Console  ConsoleConfig
	Music    MusicConfig
	Feriados FeriadosConfig

	// Owner é a identidade do dono — número e/ou JID. Substitui as duas
	// configurações antigas (usersJID.owner e owner.number), que seguem
	// sendo lidas para compatibilidade.
	Owner identity.Owner
}

type WeekdayJobConfig struct {
	Name         string   `mapstructure:"name"`
	Day          string   `mapstructure:"day"`
	Enabled      bool     `mapstructure:"enabled"`
	Hour         int      `mapstructure:"hour"`
	Minute       int      `mapstructure:"minute"`
	Message      string   `mapstructure:"message"`
	AudioPath    string   `mapstructure:"audioPath"`
	StickerName  string   `mapstructure:"stickerName"`
	TargetGroups []string `mapstructure:"targetGroups"`
}

type BotConfig struct {
	Name                   string   `mapstructure:"name"`
	Prefix                 string   `mapstructure:"prefix"`
	Environment            string   `mapstructure:"environment"`
	Timezone               string   `mapstructure:"timezone"`
	NLPGroupTrigger        bool     `mapstructure:"nlpGroupTrigger"`
	IntentEnabled          bool     `mapstructure:"intentEnabled"`
	NormalStickerChance    float64  `mapstructure:"normalStickerChance"`
	NormalStickerBlacklist []string `mapstructure:"normalStickerBlacklist"`
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

// Load carrega a configuração (YAML + env) e valida os campos obrigatórios.
// Retorna erro quando algo essencial falta — o bot não sobe em configuração
// inválida (falha no boot, não no meio do uso).
func Load() (*Config, error) {
	_ = godotenv.Load()

	// Instância própria: o viper global acumula paths e defaults entre
	// chamadas (e entre testes), o que torna o carregamento imprevisível.
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	// -1 = "não configurado" (o valor 0 é válido: desliga os stickers).
	v.SetDefault("bot.normalStickerChance", -1)

	// Defaults do subsistema de mídia.
	v.SetDefault("music.localEnabled", true)
	v.SetDefault("music.tmpDir", os.TempDir())
	v.SetDefault("music.cacheDir", "storage/music-cache")
	v.SetDefault("music.maxFileMB", 25)
	v.SetDefault("music.maxDownloads", 1)
	v.SetDefault("music.timeoutSec", 240)

	// Caminhos padrão dos binários locais.
	v.SetDefault("music.ytdlpPath", "./bin/yt-dlp")
	v.SetDefault("music.ffmpegPath", "./bin/ffmpeg")
	v.SetDefault("music.denoPath", "./bin/deno")

	// Mapeamento de env vars → chaves viper
	v.BindEnv("bot.prefix", "COMMAND_PREFIX")
	v.BindEnv("bot.environment", "ENVIRONMENT")
	v.BindEnv("log.level", "LOG_LEVEL")
	v.BindEnv("database.dsn", "DB_DSN")
	v.BindEnv("usersjid.owner", "OWNER_JID")

	// Música
	v.BindEnv("music.serverUrl", "MUSIC_SERVER_URL")
	v.BindEnv("music.apiToken", "API_AUTH_TOKEN")
	v.BindEnv("music.localEnabled", "MUSIC_LOCAL_ENABLED")
	v.BindEnv("music.ytdlpPath", "YTDLP_PATH")
	v.BindEnv("music.ffmpegPath", "FFMPEG_PATH")
	v.BindEnv("music.denoPath", "DENO_PATH")
	v.BindEnv("music.tmpDir", "MUSIC_TMP_DIR")
	v.BindEnv("music.cacheDir", "MUSIC_CACHE_DIR")
	v.BindEnv("music.maxFileMB", "MUSIC_MAX_FILE_MB")
	v.BindEnv("music.maxDownloads", "MUSIC_MAX_CONCURRENT")
	v.BindEnv("music.timeoutSec", "MUSIC_TIMEOUT_SEC")

	// APIs — path canônico (o mesmo do YAML). O path legado "apicalls.*"
	// continua lido como fallback para não quebrar quem já configurou.
	v.BindEnv("owner.number", "OWNER_NUMBER")
	v.BindEnv("feriados.apiKey", "FERIADOS_API_KEY")
	v.BindEnv("groq.url", "GROQ_URL")
	v.BindEnv("groq.apiKey", "GROQ_API_KEY")
	v.BindEnv("tavily.apiKey", "TAVILY_API_KEY")
	v.BindEnv("ia.model", "IA_MODEL")

	// Console (controle remoto dos comandos)
	v.BindEnv("console.url", "CONSOLE_URL")
	v.BindEnv("console.token", "CONSOLE_TOKEN")

	if err := v.ReadInConfig(); err != nil {
		// Sem YAML: segue só com env/default (deploy mínimo).
		log.Println("config.yaml não encontrado, usando env/default")
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("erro ao carregar config: %w", err)
	}

	// Default para timezone
	if cfg.Bot.Timezone == "" {
		cfg.Bot.Timezone = "America/Sao_Paulo"
	}

	// Default para chance de sticker aleatório. O valor -1 significa
	// "não configurado" (o YAML não traz a chave) — assim o usuário pode
	// desabilitar de verdade configurando 0.
	if cfg.Bot.NormalStickerChance < 0 {
		cfg.Bot.NormalStickerChance = 0.2
	}

	// Preenche campos que vieram de env vars mas não tem chave mapstructure no struct
	// Canonical primeiro; o path legado "apicalls.*" é o fallback (config
	// de quem rodou a versão anterior continua funcionando).
	cfg.Groq = GroqConfig{
		URL:    firstNonEmpty(v.GetString("groq.url"), v.GetString("apicalls.groq.url")),
		APIKey: firstNonEmpty(v.GetString("groq.apiKey"), v.GetString("apicalls.groq.apiKey")),
	}
	cfg.IA = IAConfig{
		Model: v.GetString("ia.model"),
	}
	cfg.Console = ConsoleConfig{
		URL:   v.GetString("console.url"),
		Token: v.GetString("console.token"),
	}
	cfg.Tavily = TavilyConfig{
		APIKey: firstNonEmpty(v.GetString("tavily.apiKey"), v.GetString("apicalls.tavily.apiKey")),
	}
	cfg.Music = MusicConfig{
		ServerURL:    v.GetString("music.serverUrl"),
		APIToken:     v.GetString("music.apiToken"),
		LocalEnabled: viper.GetBool("music.localEnabled"),
		YtdlpPath:    v.GetString("music.ytdlpPath"),
		FfmpegPath:   v.GetString("music.ffmpegPath"),
		DenoPath:     v.GetString("music.denoPath"),
		TmpDir:       v.GetString("music.tmpDir"),
		CacheDir:     v.GetString("music.cacheDir"),
		MaxFileMB:    viper.GetInt("music.maxFileMB"),
		MaxDownloads: viper.GetInt("music.maxDownloads"),
		TimeoutSec:   viper.GetInt("music.timeoutSec"),
	}
	// Identidade do dono: número vem de OWNER_NUMBER (path legado
	// "apicalls.owner.number"), JID vem de usersJID.owner (YAML ou
	// OWNER_JID). As duas formas são aceitas na comparação.
	cfg.Owner = identity.Owner{
		Number: firstNonEmpty(v.GetString("owner.number"), v.GetString("apicalls.owner.number")),
		JID:    v.GetString("usersjid.owner"),
	}
	cfg.Feriados = FeriadosConfig{
		APIKey: firstNonEmpty(v.GetString("feriados.apiKey"), v.GetString("apicalls.feriados.apiKey")),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// firstNonEmpty devolve o primeiro valor não vazio.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// validate confere os campos essenciais da configuração. Falha no boot é
// melhor do que o bot subir "saudável" e falhar no primeiro comando.
func (c *Config) validate() error {
	if c.Bot.Prefix == "" {
		return fmt.Errorf("config: bot.prefix é obrigatório (ex.: !)")
	}
	if c.Database.Dsn == "" {
		return fmt.Errorf("config: database.dsn é obrigatório")
	}
	if c.Bot.Timezone == "" {
		return fmt.Errorf("config: bot.timezone é obrigatório (ex.: America/Sao_Paulo)")
	}
	if c.Owner.Number == "" && c.Owner.JID == "" {
		return fmt.Errorf("config: owner é obrigatório (env OWNER_JID ou OWNER_NUMBER)")
	}
	if _, err := time.LoadLocation(c.Bot.Timezone); err != nil {
		return fmt.Errorf("config: bot.timezone inválido (%q): %w", c.Bot.Timezone, err)
	}
	return nil
}

// Configurações carregadas de env vars / viper (sem mapstructure no yaml)
type GroqConfig struct {
	URL    string `mapstructure:"url"`
	APIKey string `mapstructure:"apiKey"`
}

type TavilyConfig struct {
	APIKey string `mapstructure:"apiKey"`
}

// ConsoleConfig liga o bot ao Shinobu Console (controle remoto dos
// comandos). Sem URL/token, a sincronização fica desligada.
type ConsoleConfig struct {
	URL   string `mapstructure:"url"`
	Token string `mapstructure:"token"`
}

// IAConfig ajusta o comportamento dos modelos de IA.
type IAConfig struct {
	// Model sobrescreve o modelo da conversa principal. Vazio = padrão do
	// pacote (openai/gpt-oss-20b). Útil para testar personalidade
	// (qwen/qwen3.8-27b) sem recompilar.
	Model string `mapstructure:"model"`
}

type MusicConfig struct {
	ServerURL string `mapstructure:"serverUrl"`
	APIToken  string `mapstructure:"apiToken"`

	// Processamento local do yt-dlp (SquareCloud, processo filho).
	LocalEnabled bool   `mapstructure:"localEnabled"`
	YtdlpPath    string `mapstructure:"ytdlpPath"`
	FfmpegPath   string `mapstructure:"ffmpegPath"`
	DenoPath     string `mapstructure:"denoPath"`
	TmpDir       string `mapstructure:"tmpDir"`
	CacheDir     string `mapstructure:"cacheDir"`
	MaxFileMB    int    `mapstructure:"maxFileMB"`
	MaxDownloads int    `mapstructure:"maxDownloads"`
	TimeoutSec   int    `mapstructure:"timeoutSec"`
}

type OwnerConfig struct {
	Number string `mapstructure:"number"`
}

type FeriadosConfig struct {
	APIKey string `mapstructure:"apiKey"`
}

type MikaelConfig struct {
	LID    string   `mapstructure:"lid"`
	Groups []string `mapstructure:"groups"`
}
