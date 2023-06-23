package backgroundjob

import (
	"time"

	"github.com/spf13/viper"
)

const (
	// Service Name
	SvcName = "cnspec" // NOTE: this name needs to align with the service name in packages
)

type JobRunner func() error

func New() (*BackgroundScanner, error) {
	return &BackgroundScanner{}, nil
}

type BackgroundScanner struct{}

func (bs *BackgroundScanner) Run(runScanFn JobRunner) error {
	Serve(
		time.Duration(viper.GetInt64("timer"))*time.Minute,
		time.Duration(viper.GetInt64("splay"))*time.Minute,
		runScanFn)
	return nil
}
