package helpers

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	log "github.com/golang/glog"
)

var (
	ApplicationName = GetFilenameWithoutExtension(os.Args[0])
)

func GetFilenameWithoutExtension(path string) string {
	filenameWithExtension := filepath.Base(path)
	extension := filepath.Ext(filenameWithExtension)
	return strings.TrimSuffix(filenameWithExtension, extension)
}

func WaitForApplicationTermination(shutdownAction func(), applicationStopped <-chan error) error {
	stopRequested := make(chan os.Signal, 1)
	signal.Notify(stopRequested, syscall.SIGINT, syscall.SIGTERM)
	select {
	case receivedSignal := <-stopRequested:
		log.Infof("Stop signal received(%s), shutting down the server", receivedSignal.String())
		shutdownAction()
		<-applicationStopped
		return nil

	case err := <-applicationStopped:
		log.Errorf("Application stopped unexpectedly: %v", err)
		close(stopRequested)
		return err
	}
}
