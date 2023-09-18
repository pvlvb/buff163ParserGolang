package main

import (
	"buff163Parser/pkg/configManager"
	"buff163Parser/pkg/cookieParsing"
	"buff163Parser/pkg/logger"
	"buff163Parser/pkg/nonCookieParsing"
	"fmt"
	"sync"
)

func main() {
	config, err := configManager.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Error loading config: %s\n", err)
		return
	}
	//backendAPIKey := os.Getenv(config.BackendAPIKeyEnv)
	//if backendAPIKey == "" {
	//	fmt.Println("No backend API key provided!")
	//	return
	//}

	logger.Log.Info("Config was successfully loaded")

	logger.Log.Infof("%s mode was launched!", config.Mode)

	var wg sync.WaitGroup

	switch config.Mode {
	case "nonCookieParsing":
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := nonCookieParsing.StartNonCookieParsing(); err != nil {
				logger.Log.WithError(err).Errorf("Starting of nonCookieParsing failed")
			}
		}()
	case "cookieParsing":
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cookieParsing.StartCookieParsing(); err != nil {
				logger.Log.WithError(err).Errorf("Starting of cookieParsing failed")
			}
		}()
	case "fairyTale":
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := nonCookieParsing.StartNonCookieParsing(); err != nil {
				logger.Log.WithError(err).Errorf("Starting of nonCookieParsing failed")
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cookieParsing.StartCookieParsing(); err != nil {
				logger.Log.WithError(err).Errorf("Starting of cookieParsing failed")
			}
		}()
	default:
		fmt.Println("Unknown mode.")
	}
	wg.Wait()
}
