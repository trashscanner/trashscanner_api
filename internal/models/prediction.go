package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
)

type PredictionStatus string

const (
	PredictionProcessingStatus PredictionStatus = "processing"
	PredictionCompletedStatus  PredictionStatus = "completed"
	PredictionFailedStatus     PredictionStatus = "failed"
)

func (st PredictionStatus) IsValid() bool {
	switch st {
	case PredictionProcessingStatus, PredictionCompletedStatus, PredictionFailedStatus:
		return true
	}

	return false
}

func (st PredictionStatus) String() string {
	return string(st)
}

type TrashType uint8

const (
	Cardboard TrashType = iota
	Glass
	Metal
	Paper
	Plastic
	Trash
	Undefined
)

const (
	TrashTypeCardboard = "cardboard"
	TrashTypeGlass     = "glass"
	TrashTypeMetal     = "metal"
	TrashTypePaper     = "paper"
	TrashTypePlastic   = "plastic"
	TrashTypeTrash     = "trash"
	TrashTypeUndefined = "undefined"
)

func (tr TrashType) String() string {
	switch tr {
	case Cardboard:
		return TrashTypeCardboard
	case Glass:
		return TrashTypeGlass
	case Metal:
		return TrashTypeMetal
	case Paper:
		return TrashTypePaper
	case Plastic:
		return TrashTypePlastic
	case Trash:
		return TrashTypeTrash
	}

	return ""
}

func (tr TrashType) StringPtr() *string {
	str := tr.String()
	return &str
}

func NewTrashType(val string) TrashType {
	switch strings.ToLower(val) {
	case TrashTypeCardboard:
		return Cardboard
	case TrashTypeGlass:
		return Glass
	case TrashTypeMetal:
		return Metal
	case TrashTypePaper:
		return Paper
	case TrashTypePlastic:
		return Plastic
	case TrashTypeTrash:
		return Trash
	}

	return Undefined
}

type PredictionResult map[string]float64

func NewPredictionResult(m map[uint8]float64) PredictionResult {
	result := make(PredictionResult, len(m))
	for k, v := range m {
		trashType := TrashType(k)
		result[trashType.String()] = v
	}
	return result
}

type Prediction struct {
	ID        uuid.UUID        `json:"id"`
	UserID    uuid.UUID        `json:"user_id"`
	TrashScan string           `json:"scan_key"`
	Status    PredictionStatus `json:"status"`
	Result    PredictionResult `json:"result"`
	Error     string           `json:"error"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (pr Prediction) IsValid() bool {
	return pr.Status.IsValid()
}

func (pr *Prediction) Model(prediction db.Prediction) {
	pr.ID = prediction.ID
	pr.UserID = prediction.UserID
	pr.TrashScan = prediction.TrashScan
	pr.Status = PredictionStatus(prediction.Status)
	pr.Result = nil
	if prediction.Result != nil {
		_ = json.Unmarshal(prediction.Result, &pr.Result)
	}
	if prediction.Error != nil {
		pr.Error = *prediction.Error
	}
	pr.CreatedAt = prediction.CreatedAt
	pr.UpdatedAt = prediction.UpdatedAt
}

func NewPredictionsList(predictions []db.Prediction) []*Prediction {
	models := make([]*Prediction, len(predictions))
	for i, dbPr := range predictions {
		model := &Prediction{}
		model.Model(dbPr)
		models[i] = model
	}

	return models
}
