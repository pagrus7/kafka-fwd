package config

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func ProvideLogger(viper *viper.Viper) (*zap.Logger, error) {
	logConf := zap.NewDevelopmentConfig()
	logConf.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger, err := logConf.Build()
	if err != nil {
		return nil, err
	}

	logger.Info("Logging initialized")
	return logger, nil
}

func ProvideSugaredLogger(logger *zap.Logger) *zap.SugaredLogger {
	return logger.Sugar()
}
