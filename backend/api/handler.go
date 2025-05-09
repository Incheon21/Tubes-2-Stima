package api

import (
	"backend/model"
)

type Handler struct {
	elements map[string]model.Element
}

func NewHandler(elements map[string]model.Element) *Handler {
	return &Handler{elements: elements}
}
