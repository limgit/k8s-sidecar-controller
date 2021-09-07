package main

import "github.com/sirupsen/logrus"

var (
	logger *logrus.Entry
)

func getLogger() *logrus.Entry {
	return logger
}

func initLogger() {
	logger = logrus.NewEntry(logrus.New())
	logger.Logger.SetFormatter(&logrus.JSONFormatter{})
	logger.Logger.SetLevel(logrus.DebugLevel)
}
