package main

import (
	// personal packages
	"github.com/fredix/fred/fred/fredlog"
	"github.com/fredix/fred/fred/fredutils"

	// standard packages
	"fmt"
	"log"
	"os"

	// external packages
	"github.com/BurntSushi/toml"
	"github.com/kardianos/service"
)

// Program structures.
//  Define Start and Stop methods.
type program struct {
	exit chan struct{}
}

type tomlConfig struct {
	Title    string
	Fred     fredlog.FredConfig
	Graylog  fredutils.Graylog
	Watchers map[string]fredutils.Watcher
}

var ConfigFile string = "fred.toml"
var logger service.Logger
var MaxTaches int = 5

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.

	if service.Interactive() {
		logger.Info("Running in terminal.")
	} else {
		logger.Info("Running under service manager.")
	}
	p.exit = make(chan struct{})

	go p.run()
	return nil
}

func (p *program) run() {
	// Do work here

	var config tomlConfig
	if _, err := toml.DecodeFile(ConfigFile, &config); err != nil {
		panic(fmt.Sprintf("%s", err))
	}

	for watcherName, watcher := range config.Watchers {

		fredlog.PrintLog(&config.Fred, fmt.Sprintf("Watcher: name => '%s' type => '%s' directory => '%s', files extension => '%s')\n", watcherName, watcher.Watcher_type, watcher.Directory, watcher.Ext_file))

		if watcher.Watcher_type == "event" {
			fredlog.PrintLog(&config.Fred, fmt.Sprintln("watcher type EVENT : ", watcher.Watcher_type))
			// launch watchers
			go fredutils.LogNewWatcher(&config.Fred, &config.Graylog, watcher)
		} else if watcher.Watcher_type == "loop" {
			fredutils.LoopDirectory(&config.Fred, &config.Graylog, watcher)
		}

	}
	//       <-done

}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	logger.Info("I'm Stopping!")
	close(p.exit)
	return nil
}

func main() {

	if len(os.Args) > 1 {
		ConfigFile = os.Args[1]
	}

	svcConfig := &service.Config{
		Name:        "fred",
		DisplayName: "fred Service",
		Description: "Files Removal Enforced Daemon",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
