package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeConfig *rest.Config
)

func getKubeConfig() *rest.Config {
	return kubeConfig
}

func initKubeClient() kubernetes.Interface {
	var err error
	config := getConfig()

	if config.KubeConfigFile != "" { // Use given config file
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", config.KubeConfigFile)
	} else { // Use in-cluster config
		kubeConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		panic(err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	return kubeClient
}
