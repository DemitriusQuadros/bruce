package pages

import (
	"fmt"
	"bruce/app/entities"
	dependencies "bruce/cmd/console/dependencies"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type DummyUseCase interface {
	GetAllDummies() ([]entities.Dummy, error)
}

type DummyPage struct {
	List    *widgets.List
	UseCase DummyUseCase
}

func NewDummyPage(uc DummyUseCase) *DummyPage {
	l := widgets.NewList()
	l.Title = " Dummy Entities "
	l.TextStyle = ui.NewStyle(ui.ColorYellow)
	l.WrapText = false
	return &DummyPage{
		List:    l,
		UseCase: uc,
	}
}

func (p *DummyPage) Render(width, height int) *widgets.List {
	p.List.SetRect(0, 3, width, height)
	p.fetchDummies()
	return p.List
}

func (p *DummyPage) Set(header *widgets.Paragraph, tabPane *widgets.TabPane, _ *dependencies.Dependencies) *DummyPage {
	// Optional setup configuration
	return p
}

func (p *DummyPage) fetchDummies() {
	dummies, err := p.UseCase.GetAllDummies()
	if err != nil {
		p.List.Rows = []string{fmt.Sprintf("Error fetching dummies: %v", err)}
		return
	}

	if len(dummies) == 0 {
		p.List.Rows = []string{"No dummy entities found."}
		return
	}

	var rows []string
	rows = append(rows, fmt.Sprintf("%-10s | %-30s", "ID", "TEXT"))
	rows = append(rows, "---------------------------------------------")
	for _, d := range dummies {
		rows = append(rows, fmt.Sprintf("%-10d | %-30s", d.ID, d.Text))
	}
	p.List.Rows = rows
}
