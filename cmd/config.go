package main

import (
	"flag"
	"os"
	"os/user"
	"strings"
)

var (
	config *Config
)

type Config struct {
	KubeConfigFile string
}

func getConfig() *Config {
	return config
}

func initConfig() {
	config = new(Config)

	// Parse commandline arguments
	flag.StringVar(&config.KubeConfigFile, "kubeconfig", "", "path to kubeconfig file")

	flag.Parse()

	if config.KubeConfigFile == "" {
		config.KubeConfigFile = os.Getenv("KUBECONFIG") // Fallback to env if not given
	}
	if strings.Contains(config.KubeConfigFile, "~") {
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}
		config.KubeConfigFile = strings.Replace(config.KubeConfigFile, "~", usr.HomeDir, -1)
	}
}
