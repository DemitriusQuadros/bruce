package main

import (
	"log"

	repository "bruce/app/repository/dummy"
	service "bruce/app/services/dummy"
	usecase "bruce/app/usecase/dummy"
	dependencies "bruce/cmd/console/dependencies"
	"bruce/cmd/console/pages"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func main() {
	deps := dependencies.Init()

	// Initialize the Dummy layer locally for the console
	dummyRepo := repository.NewDummyRepository(deps.Db)
	dummySvc := service.NewDummyService()
	dummyUC := usecase.NewDummyUseCase(dummyRepo, dummySvc)

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	termWidth, termHeight := ui.TerminalDimensions()

	header := widgets.NewParagraph()
	header.Text = "Press ESC to quit"
	header.SetRect(0, 0, termWidth, 1)
	header.Border = false
	header.TextStyle.Bg = ui.ColorBlue

	tabPane := widgets.NewTabPane("Dummies")
	tabPane.SetRect(0, 1, termWidth, 2)
	tabPane.Border = true
	tabPane.BorderStyle.Fg = ui.ColorGreen

	dummyPage := pages.NewDummyPage(dummyUC)

	ui.Render(header, tabPane, dummyPage.Render(termWidth, termHeight))

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "<Escape>", "q", "<C-c>":
			return
		case "<Resize>":
			payload := e.Payload.(ui.Resize)
			termWidth, termHeight = payload.Width, payload.Height
			header.SetRect(0, 0, termWidth, 1)
			tabPane.SetRect(0, 1, termWidth, 2)
			ui.Clear()
			ui.Render(header, tabPane, dummyPage.Render(termWidth, termHeight))
		}
	}
}
