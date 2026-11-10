// Orquestração do download de música para o comando !play.
//
// Prioridade do fluxo:
//  1. cache em disco (TTL 7d) — evita refazer busca de áudio já baixado
//  2. execução local do yt-dlp (Executor, processo filho, SquareCloud)
//  3. fallback remoto (notebook via túnel MUSIC_SERVER_URL) — legado,
//     mantido só enquanto a migração local estiver em observação
//
// O Service não conhece WhatsApp nem comandos — devolve bytes + extensão.
package music

import (
	"context"
	"fmt"
	"os"
)

// Service coordena cache, executor local e fallback remoto.
type Service struct {
	exec      *Executor // nil = processamento local desativado
	cache     *Cache    // nil = cache desativado
	serverURL string    // fallback remoto (túnel)
	apiToken  string
}

// NewService monta o Service. executor e cache podem ser nil (desativados).
func NewService(exec *Executor, cache *Cache, serverURL, apiToken string) *Service {
	return &Service{exec: exec, cache: cache, serverURL: serverURL, apiToken: apiToken}
}

// DownloadAudio resolve a query e devolve os bytes do áudio + extensão.
// Extensão é sempre "m4a" nos fluxos locais; o túnel remoto devolve "mp3".
func (s *Service) DownloadAudio(ctx context.Context, query string) ([]byte, string, error) {
	// 1. Cache.
	if s.cache != nil {
		s.cache.Sweep()
		if p := s.cache.Get(query); p != "" {
			data, err := os.ReadFile(p)
			if err == nil {
				return data, "m4a", nil
			}
			// Entrada de cache corrompida/inacessível: segue pra execução local.
		}
	}

	// 2. Execução local (yt-dlp como processo filho).
	if s.exec != nil {
		data, err := s.downloadLocal(ctx, query)
		if err == nil {
			return data, "m4a", nil
		}
		// Falha classificada (origem recusou etc): propaga — o túnel
		// rodaria contra a mesma origem e não resolveria.
		if ed, ok := AsErrDownload(err); ok && ed.Category != ErrUnknown {
			return nil, "", err
		}
		// Falha de infraestrutura local (binário ausente etc): tenta túnel.
	}

	// 3. Fallback remoto (notebook via túnel) quando local falhou/desativado.
	if s.serverURL != "" && s.apiToken != "" {
		return s.downloadViaTunnel(ctx, query)
	}

	if s.exec == nil {
		return nil, "", &ErrDownload{Category: ErrUnknown,
			Detail: "nenhum método disponível: processamento local desativado e MUSIC_SERVER_URL não configurado"}
	}
	// Local ativo, mas com falha não classificada e sem fallback.
	return nil, "", &ErrDownload{Category: ErrUnknown, Detail: "execução local falhou sem categoria"}
}

// downloadLocal roda o executor e lê o resultado. O destino é o cache
// (se ativo) ou um temporário removido após a leitura.
func (s *Service) downloadLocal(ctx context.Context, query string) ([]byte, error) {
	dest, ok := s.cacheDest(query)
	if !ok {
		var err error
		if dest, err = tempPath(); err != nil {
			return nil, err
		}
		defer os.Remove(dest)
	}

	if err := s.exec.Download(ctx, query, dest); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		return nil, fmt.Errorf("music/service: ler resultado: %w", err)
	}
	return data, nil
}

// cacheDest devolve (caminho, true) se há cache ativo.
func (s *Service) cacheDest(query string) (string, bool) {
	if s.cache == nil {
		return "", false
	}
	return s.cache.Put(query), true
}

// tempPath cria um temporário para o áudio quando o cache está desativado.
func tempPath() (string, error) {
	f, err := os.CreateTemp("", "shinobu-audio-*.m4a")
	if err != nil {
		return "", fmt.Errorf("music/service: temporário: %w", err)
	}
	f.Close()
	return f.Name(), nil
}

// AsErrDownload extrai *ErrDownload de erros estaticamente tipados.
func AsErrDownload(err error) (*ErrDownload, bool) {
	ed, ok := err.(*ErrDownload)
	return ed, ok
}
