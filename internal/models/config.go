package models

import (
	"time"

	"github.com/sirupsen/logrus"
)

type Config struct {
	Name        string `json:"name"`
	ShortDesc   string `json:"short-desc"`
	LongDesc    string `json:"long-desc"`
	Version     string `json:"version"`
	TimeUnit    string `json:"timeunit"`
	TimeoutInt  int    `json:"timeout"`
	Logger      *logrus.Logger
	Commands    []*Command
	ThreadCount int
	UsePipeline bool
	Verbose     bool
	Timeout     time.Duration
}
