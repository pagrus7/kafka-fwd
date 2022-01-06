package config

import (
	"context"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// syslogPriority is used to convert by kafka log event level into appropriate logger methof
type syslogPriority int

// Constants per syslog.h
const (
	LogEmerg syslogPriority = iota
	LogAlert
	LogCrit
	LogErr
	LogWarning
	LogNotice
	LogInfo
	LogDebug
)

func forwardLogEvents(logger *zap.SugaredLogger, logChan *chan kafka.LogEvent) {
	for e := range *logChan {
		logFunc := logger.Info
		switch syslogPriority(e.Level) {
		case LogEmerg:
			logFunc = logger.Panic
		case LogErr, LogAlert, LogCrit:
			logFunc = logger.Error
		case LogWarning:
			logFunc = logger.Warn
		case LogInfo, LogNotice:
			logFunc = logger.Info
		case LogDebug:
			logFunc = logger.Debug
		}

		logFunc(e.Name + ": " + e.Message)
	}
}

type KafkaContext struct {
	Consumer *kafka.Consumer
	Producer *kafka.Producer
	logger   *zap.SugaredLogger
}

func ProvideKafkaContext(viper *viper.Viper, logger *zap.SugaredLogger, lc fx.Lifecycle) (*KafkaContext, error) {
	var prodConf *kafka.ConfigMap
	var consConf *kafka.ConfigMap
	var consumer *kafka.Consumer
	var producer *kafka.Producer
	var err error

	if prodConf, consConf, err = buildKafkaConfig(viper, logger); err != nil {
		return nil, err
	}

	if producer, err = buildKafkaProducer(prodConf, logger); err != nil {
		return nil, err
	}

	if consumer, err = buildKafkaConsumer(consConf, logger); err != nil {

		return nil, err
	}

	kCtx := KafkaContext{
		Consumer: consumer,
		Producer: producer,
		logger:   logger.Named("kafka-context"),
	}

	stopHook := fx.Hook{
		OnStop: kCtx.fxClose,
	}
	lc.Append(stopHook)
	return &kCtx, nil
}

func (kc *KafkaContext) fxClose(ctx context.Context) error {
	return kc.Close()
}

func (kc *KafkaContext) Close() error {
	kc.logger.Infof("Closing consumer %s", kc.Consumer)
	if err := kc.Consumer.Close(); err != nil {
		return err
	}
	kc.logger.Info("Consumer closed")
	kc.logger.Infof("Closing producer %s", kc.Producer)
	kc.Producer.Close()
	kc.logger.Info("Producer closed")
	return nil
}

func buildKafkaConfig(viper *viper.Viper, logger *zap.SugaredLogger) (prodCfg *kafka.ConfigMap, consCfg *kafka.ConfigMap, err error) {
	logger.Debug("Building kafka config")

	prodConf, consConf := kafka.ConfigMap{}, kafka.ConfigMap{}

	logger.Debug("Building producer config")
	if err := applyToConfig(viper.Sub("kafka.common"), &prodConf, logger); err != nil {
		return nil, nil, err
	}

	if err := applyToConfig(viper.Sub("kafka.producer"), &prodConf, logger); err != nil {
		return nil, nil, err
	}

	logger.Debug("Building consumer config")
	if err := applyToConfig(viper.Sub("kafka.common"), &consConf, logger); err != nil {
		return nil, nil, err
	}
	if err := applyToConfig(viper.Sub("kafka.consumer"), &consConf, logger); err != nil {
		return nil, nil, err
	}

	// mandatory producer params
	logChan := make(chan kafka.LogEvent)
	_ = prodConf.SetKey("go.logs.channel.enable", true)
	_ = prodConf.SetKey("go.logs.channel", logChan)
	_ = prodConf.SetKey("enable.idempotence", true)

	// mandatory consumer params
	_ = consConf.SetKey("go.logs.channel.enable", true)
	_ = consConf.SetKey("go.logs.channel", logChan)
	_ = consConf.SetKey("enable.auto.commit", true)
	_ = consConf.SetKey("enable.auto.offset.store", false)

	go forwardLogEvents(logger.Named("kafka-log-events"), &logChan)
	return &prodConf, &consConf, nil
}

func applyToConfig(viper *viper.Viper, conf *kafka.ConfigMap, logger *zap.SugaredLogger) error {
	if viper == nil {
		return nil
	}
	for _, key := range viper.AllKeys() {
		logger.Debug("Applying key ", key)
		if err := conf.SetKey(key, viper.GetString(key)); err != nil {
			return err
		}
	}
	return nil
}

const bootstrapServersKey = "bootstrap.servers"

func buildKafkaConsumer(conf *kafka.ConfigMap, logger *zap.SugaredLogger) (*kafka.Consumer, error) {
	servers, _ := conf.Get(bootstrapServersKey, "UNDEFINED")
	logger.Infof("Initializing kafka consumer for %s", servers)

	return kafka.NewConsumer(conf)
}

func buildKafkaProducer(conf *kafka.ConfigMap, logger *zap.SugaredLogger) (*kafka.Producer, error) {
	servers, _ := conf.Get("bootstrap.servers", "UNDEFINED")
	logger.Infof("Initializing kafka producer for %s", servers)

	return kafka.NewProducer(conf)
}
