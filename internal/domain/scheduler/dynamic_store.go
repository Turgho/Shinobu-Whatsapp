package scheduler

import (
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/store"
	"go.uber.org/zap"
)

// DataWithTime associa um DynamicJobData ao seu RunAt já parseado.
type DataWithTime struct {
	Data     DynamicJobData
	ParsedAt time.Time
}

type DynamicJobData struct {
	ID         string `json:"id"`
	RunAt      string `json:"run_at"`
	ChatJID    string `json:"chat_jid"`
	Message    string `json:"message"`
	MentionAll bool   `json:"mention_all,omitempty"`
}

type DynamicStore struct {
	js     *store.JSONStore[[]DynamicJobData]
	logger *zap.Logger
}

func NewDynamicStore(path string, logger *zap.Logger) *DynamicStore {
	return &DynamicStore{
		js:     store.NewJSONStore[[]DynamicJobData](path),
		logger: logger,
	}
}

func (ds *DynamicStore) Save(job *DynamicJob) error {
	data := DynamicJobData{
		ID:         job.ID,
		RunAt:      job.RunAt.Format(time.RFC3339),
		ChatJID:    job.ChatJID,
		Message:    job.Message,
		MentionAll: job.MentionAll,
	}

	return ds.js.Update(func(list []DynamicJobData) ([]DynamicJobData, error) {
		if list == nil {
			list = make([]DynamicJobData, 0)
		}
		list = append(list, data)
		return list, nil
	})
}

func (ds *DynamicStore) Delete(id string) error {
	return ds.js.Update(func(list []DynamicJobData) ([]DynamicJobData, error) {
		if list == nil {
			return list, nil
		}
		for i, d := range list {
			if d.ID == id {
				list = append(list[:i], list[i+1:]...)
				break
			}
		}
		return list, nil
	})
}

func (ds *DynamicStore) LoadAll() []DynamicJobData {
	list, err := ds.js.Read()
	if err != nil || list == nil {
		ds.logger.Warn("Erro ao carregar jobs dinâmicos", zap.Error(err))
		return nil
	}
	return list
}
