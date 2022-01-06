package config

import (
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
)

const (
	envPrefix         = "KFWD_"
	configType        = "yaml"
	configLocsKey     = "configLocs"
	defaultConfigLocs = ".:./conf"
	configFileKey     = "configFile"
	defaultConfigName = "kfwd-conf"
)

func ProvideViper() (*viper.Viper, error) {
	v := viper.New()

	configLocs := getOrDefault(v, configLocsKey, defaultConfigLocs)
	for _, loc := range strings.Split(configLocs, ":") {
		log.Println("Registering config location ", loc)
		v.AddConfigPath(loc)
	}

	configName := getOrDefault(v, configFileKey, defaultConfigName)
	log.Printf("Reading config %s of type %s", configName, configType)
	v.SetConfigName(configName)
	v.SetConfigType(configType)
	v.SetConfigFile("")
	err := v.ReadInConfig()
	if err != nil {
		log.Println("Error: Could not read config file: ", err)
		return nil, err
	}

	// manually publish env variables prefixed with KFWD_
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, envPrefix) {
			parts := strings.Split(env, "=")
			key := strings.TrimPrefix(parts[0], envPrefix)
			key = strings.ToLower(key)
			key = strings.ReplaceAll(key, "_", ".")
			log.Printf("Importing %s from env", key)
			v.Set(key, parts[1])
		}
	}

	return v, nil
}

func getOrDefault(v *viper.Viper, key string, defVal string) string {
	if v.IsSet(key) {
		return v.GetString(key)
	}
	return defVal
}
