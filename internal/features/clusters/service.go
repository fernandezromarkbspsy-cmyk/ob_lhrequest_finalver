package clusters

import (
	"net/http"
	"time"
)

type Option struct {
	ID        uint   `json:"id"`
	Cluster   string `json:"cluster"`
	HubName   string `json:"hub_name"`
	Region    string `json:"region"`
	DockNo    string `json:"dock_no"`
	Backlogs  int    `json:"backlogs"`
	BacklogTS string `json:"backlogs_ts"`
}

type AppError struct {
	Code    int
	Message string
}

func (e AppError) Error() string {
	return e.Message
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) List() ([]Option, error) {
	options := []Option{}
	if !s.repo.Available() {
		return options, nil
	}
	records, err := s.repo.List()
	if err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to load clusters"}
	}
	seen := map[string]bool{}
	for _, record := range records {
		key := record.ClusterName + "|" + record.HubName + "|" + record.Region + "|" + record.DockNumber
		if seen[key] {
			continue
		}
		seen[key] = true
		backlogsTS := ""
		if record.BacklogsTS != nil {
			backlogsTS = record.BacklogsTS.Format(time.RFC3339)
		}
		options = append(options, Option{
			ID:        record.ID,
			Cluster:   record.ClusterName,
			HubName:   record.HubName,
			Region:    record.Region,
			DockNo:    record.DockNumber,
			Backlogs:  record.Backlogs,
			BacklogTS: backlogsTS,
		})
	}
	return options, nil
}
