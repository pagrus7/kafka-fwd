package main

import (
	"github.com/pagrus7/kafka-fwd/config"
	"github.com/pagrus7/kafka-fwd/routing"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		fx.Provide(config.ProvideLogger),
		fx.Provide(config.ProvideSugaredLogger),
		fx.Provide(config.ProvideViper),
		fx.Provide(config.ProvideRouteConfig),
		fx.Provide(config.ProvideKafkaContext),
		fx.Provide(routing.ProvideRouter),
		fx.Invoke(Register),
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger}
		}),
	)

	app.Run()
}

func Register(r *routing.Router) {}
