package main

import (
	// internal package
	"fredutils"

	// standard packages
	"fmt"
	"log"
	"os"
	"time"

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
	Owner    ownerInfo
	Graylog  fredutils.Graylog
	Watchers map[string]fredutils.Dir_watcher
}

type ownerInfo struct {
	Name string
	Org  string `toml:"organization"`
	DOB  time.Time
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

	fmt.Println("Title: ", config.Title)
	fmt.Printf("Owner: %s (%s, %s)\n",
		config.Owner.Name, config.Owner.Org, config.Owner.DOB)

	fmt.Printf("Graylog: %s %d (Format %s) (Protocol %s)\n",
		config.Graylog.Ip, config.Graylog.Port, config.Graylog.Format,
		config.Graylog.Protocol)

	fmt.Println("Watchers")

	//        done := make(chan bool)

	for watcherName, dir_watcher := range config.Watchers {
		fmt.Printf("Watcher: %s (%s, %s, %s %s %d)\n", watcherName, dir_watcher.Directory, dir_watcher.Ext_file, dir_watcher.Payload_host, dir_watcher.Payload_level)

		fmt.Println("watcher type : ", dir_watcher.Watcher_type)

		if dir_watcher.Watcher_type == "event" {
			fmt.Println("watcher type EVENT : ", dir_watcher.Watcher_type)
			// launch watchers
			go fredutils.LogNewWatcher(&config.Graylog, dir_watcher)
		} else if dir_watcher.Watcher_type == "loop" {
			fredutils.LoopDirectory(&config.Graylog, dir_watcher)
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
