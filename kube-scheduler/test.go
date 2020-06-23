package main

import (

	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
	"kube-scheduler/myfilter"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	command := app.NewSchedulerCommand(
		// WithPlugin : https://github.com/kubernetes/kubernetes/blob/master/cmd/kube-scheduler/app/server.go#L302
		app.WithPlugin(myfilter.PluginName, myfilter.New ),
	)

	log.Debug("Starting the uds scheduler plugins")
	if err := command.Execute(); err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to run uds scheduler plugins")
		os.Exit(1)
	}
}