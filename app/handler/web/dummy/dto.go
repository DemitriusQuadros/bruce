package handler

import "bruce/app/entities"

type CreateDummyDto struct {
	Text string `json:"text"`
}

func (dto CreateDummyDto) ToModel() entities.Dummy {
	return entities.Dummy{
		Text: dto.Text,
	}
}

type DummyResponseDto struct {
	ID   int64  `json:"id"`
	Text string `json:"text"`
}

func FromModel(dummy entities.Dummy) DummyResponseDto {
	return DummyResponseDto{
		ID:   dummy.ID,
		Text: dummy.Text,
	}
}
