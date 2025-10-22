package models

import (
	"errors"
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

func (tr TrashType) String() string {
	switch tr {
	case Cardboard:
		return "cardboard"
	case Glass:
		return "glass"
	case Metal:
		return "metal"
	case Paper:
		return "paper"
	case Plastic:
		return "plastic"
	case Trash:
		return "trash"
	}

	return ""
}

func (tr TrashType) StringPtr() *string {
	str := tr.String()
	return &str
}

func NewTrashType(val string) TrashType {
	switch strings.ToLower(val) {
	case "cardboard":
		return Cardboard
	case "glass":
		return Glass
	case "metal":
		return Metal
	case "paper":
		return Paper
	case "plastic":
		return Plastic
	case "trash":
		return Trash
	}

	return Undefined
}

type Prediction struct {
	ID         uuid.UUID
	User_id    uuid.UUID
	Trash_scan string
	Status     PredictionStatus
	Result     TrashType
	Error      error

	CreatedAt, UpdatedAt time.Time
}

func (pr Prediction) IsValid() bool {
	return pr.Status.IsValid()
}

func (pr *Prediction) Model(prediction db.Prediction) {
	pr.ID = prediction.ID
	pr.User_id = prediction.UserID
	pr.Trash_scan = prediction.TrashScan
	pr.Status = PredictionStatus(prediction.Status)
	pr.Result = Undefined
	if prediction.Result != nil {
		pr.Result = NewTrashType(*prediction.Result)
	}
	if prediction.Error != nil {
		pr.Error = errors.New(*prediction.Error)
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
