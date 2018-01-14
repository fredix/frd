package main

import (
	// personal packages
	"github.com/fredix/frd/frd/frdlog"
	"github.com/fredix/frd/frd/frdutils"

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
	Frd      frdlog.FrdConfig
	Graylog  frdutils.Graylog
	Watchers map[string]frdutils.Watcher
}

var ConfigFile string = "frd.toml"
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

		frdlog.PrintLog(&config.Frd, fmt.Sprintf("Watcher: name => '%s' type => '%s' directory => '%s', files extension => '%s')\n", watcherName, watcher.Watcher_type, watcher.Directory, watcher.Ext_file))

		if watcher.Watcher_type == "event" {
			frdlog.PrintLog(&config.Frd, fmt.Sprintln("watcher type EVENT : ", watcher.Watcher_type))
			// launch watchers
			go frdutils.LogNewWatcher(&config.Frd, &config.Graylog, watcher)
		} else if watcher.Watcher_type == "loop" {
			frdutils.LoopDirectory(&config.Frd, &config.Graylog, watcher)
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
		Name:        "frd",
		DisplayName: "frd Service",
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
