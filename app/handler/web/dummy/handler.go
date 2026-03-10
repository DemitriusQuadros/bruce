package handler

import (
	"encoding/json"
	"bruce/app/entities"
	"bruce/internal/handler"
	"net/http"
	"strconv"
)

type UseCase interface {
	CreateDummy(text string) error
	GetDummy(id int64) (entities.Dummy, error)
	ProcessDummy(id int64) error
	GetAllDummies() ([]entities.Dummy, error)
}

type DummyHandler struct {
	UseCase UseCase
}

func NewDummyHandler(u UseCase) *DummyHandler {
	return &DummyHandler{
		UseCase: u,
	}
}

func (h *DummyHandler) Handlers() []handler.Configuration {
	return []handler.Configuration{
		{
			Pattern: "/dummy",
			Action:  h.Post,
			Method:  http.MethodPost,
		},
		{
			Pattern: "/dummy",
			Action:  h.Get,
			Method:  http.MethodGet,
		},
		{
			Pattern: "/dummy/process",
			Action:  h.Process,
			Method:  http.MethodPost,
		},
		{
			Pattern: "/dummy/all",
			Action:  h.GetAll,
			Method:  http.MethodGet,
		},
	}
}

func (h *DummyHandler) Post(w http.ResponseWriter, r *http.Request) {
	var body CreateDummyDto
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.UseCase.CreateDummy(body.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *DummyHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dummy, err := h.UseCase.GetDummy(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(FromModel(dummy)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *DummyHandler) Process(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.UseCase.ProcessDummy(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *DummyHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	dummies, err := h.UseCase.GetAllDummies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var response []DummyResponseDto
	for _, d := range dummies {
		response = append(response, FromModel(d))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
