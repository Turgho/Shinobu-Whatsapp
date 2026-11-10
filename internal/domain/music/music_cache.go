// Cache de áudio por consulta (reaproveita ideia do servidor remoto: TTL 7 dias).
//
// Design mínimo proposital:
//   - chave = SHA-256 da query normalizada + formato
//   - valor = arquivo m4a em CacheDir
//   - expiração checada na leitura; sweep limpa vencidos no início de cada busca
//   - sem índice em disco: o nome do arquivo É a chave
package music

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// CacheTTL varredura mínima entre limpezas de arquivos vencidos.
const CacheTTL = 24 * time.Hour

// Cache é o cache em disco de áudios prontos.
type Cache struct {
	dir string
}

// NewCache cria o diretório do cache. Retorna nil se desativado (dir vazio).
// Falhas de mkdir não derrubam o bot — só logam e desativam.
func NewCache(dir string, logger *zap.Logger) *Cache {
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		if logger != nil {
			logger.Warn("music/cache: diretório inacessível, cache desativado", zap.String("dir", dir), zap.Error(err))
		}
		return nil
	}
	return &Cache{dir: dir}
}

// cacheKey deriva nome de arquivo determinístico da query.
func cacheKey(query string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(
		strings.TrimPrefix(strings.TrimPrefix(query, "http://"), "https://")))))
	return hex.EncodeToString(h[:16]) + ".m4a"
}

// Get devolve o caminho em cache se existir e não estiver vencido.
// Remove o arquivo vencido em disco antes de retornar miss.
func (c *Cache) Get(query string) string {
	if c == nil {
		return ""
	}
	path := filepath.Join(c.dir, cacheKey(query))
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return ""
	}
	// TTL: mod-time é a data do download.
	if time.Since(info.ModTime()) > CacheTTL {
		os.Remove(path)
		return ""
	}
	return path
}

// Put devolve o caminho onde o resultado da query deve ficar.
// O download roda em tmpdir e o chamador renomeia pra cá em caso de sucesso.
func (c *Cache) Put(query string) string {
	if c == nil {
		return ""
	}
	return filepath.Join(c.dir, cacheKey(query))
}

// Sweep remove arquivos vencidos. É barata (dir pequeno) e executada
// no início de cada busca — sem goroutine extra.
func (c *Cache) Sweep() {
	if c == nil {
		return
	}
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-CacheTTL)
	for _, e := range entries {
		full := filepath.Join(c.dir, e.Name())
		if info, err := e.Info(); err == nil && info.ModTime().Before(cutoff) {
			os.Remove(full)
		}
	}
}

// Stats devolve quantidade e tamanho do cache para o comando admin/stats.
func (c *Cache) Stats() (entries int, sizeMB float64) {
	if c == nil {
		return 0, 0
	}
	d, err := os.ReadDir(c.dir)
	if err != nil {
		return 0, 0
	}
	for _, e := range d {
		if info, err := e.Info(); err == nil {
			entries++
			sizeMB += float64(info.Size())
		}
	}
	return entries, sizeMB / 1024 / 1024
}
