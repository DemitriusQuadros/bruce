package modules

import (
	taskHandler "bruce/app/handler/tasks/dummy"
	webHandler "bruce/app/handler/web/dummy"
	repository "bruce/app/repository/dummy"
	service "bruce/app/services/dummy"
	usecase "bruce/app/usecase/dummy"
	worker "bruce/app/workers/dummy"

	"go.uber.org/fx"
)

var DummyModule = fx.Module("dummy",
	fx.Provide(
		repository.NewDummyRepository,
		service.NewDummyService,
		usecase.NewDummyUseCase,
		worker.NewDummyWorker,
		taskHandler.NewDummyProcessor,
		webHandler.NewDummyHandler,
		func(s repository.DummyRepository) usecase.DummyRepository { return s },
		func(s service.DummyService) usecase.DummyService { return s },
		func(s *usecase.DummyUseCase) taskHandler.DummyUseCase { return s },
		func(s *usecase.DummyUseCase) webHandler.UseCase { return s },
	),
)
