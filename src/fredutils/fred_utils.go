package fredutils

import (
	// standard packages
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	// external packages
	//"github.com/Graylog2/go-gelf/gelf"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/Graylog2/go-gelf.v2/gelf"
)

type Message struct {
	ID          int
	Version     string
	Environment string
	Message     string `json:"short_message"`
	Host        string `json:"host"`
	Level       int    `json:"level"`
	MessageLog  string `json:"_log"`
	File        string `json:"_file"`
	Localtime   string `json:"_localtime"`
}

type Graylog struct {
	Ip       string `toml:"ip"`
	Port     int    `toml:"port"`
	Format   string `toml:"format"`
	Protocol string `toml:"protocol"`
}

type Dir_watcher struct {
	Watcher_type  string
	Name          string
	Environment   string
	Directory     string
	Ext_file      string
	Payload_host  string
	Payload_level int
	Removetime    string
	Loop_sleep    string
}

func LoopDirectory(graylog *Graylog, dir_watcher Dir_watcher) {
	go func() {
		for {
			ListFilesAndPush(graylog, dir_watcher)
			//time.Sleep(60000 * time.Millisecond)
			fmt.Println("dir_watcher.Loop_sleep : ", dir_watcher.Loop_sleep)
			fmt.Println("dir_watcher.Ext_file : ", dir_watcher.Ext_file)
			duration, _ := time.ParseDuration(dir_watcher.Loop_sleep)
			fmt.Println("Duration sleep : ", duration)
			time.Sleep(duration)
		}
	}()
}

func ListFilesAndPush(graylog *Graylog, dir_watcher Dir_watcher) {
	ip := graylog.Ip + ":" + strconv.Itoa(graylog.Port)

	files, _ := ioutil.ReadDir(dir_watcher.Directory)
	for _, f := range files {
		file_ext := filepath.Ext(f.Name())
		if file_ext == dir_watcher.Ext_file {
			fmt.Println("file to push : ", dir_watcher.Directory+"/"+f.Name())
			payload := payload(dir_watcher.Environment, dir_watcher.Name, f.Name(), dir_watcher.Directory+"/"+f.Name(), dir_watcher.Payload_host, dir_watcher.Payload_level)
			PushToGraylogUdp(
				dir_watcher,
				ip,
				&payload)
		}
	}
}

/*
func payload(environment string, msg string, messagelog string, file string, host string, level int) Message {

	t := time.Now()
	m := Message{
		Version:    "1.1",
		Environment: environment,
		Message:    msg,
		Host:       host,
		Level:      level,
		MessageLog: messagelog,
		File:       file,
		Localtime:  t.Format("01-02-2006T15-04-05"),
	}



	return m
}
*/

func payload(environment string, msg string, messagelog string, file string, host string, level int) gelf.Message {

	// get last modified time
	filename, err := os.Stat(file)
	if err != nil {
		fmt.Println(err)
	}
	filetime := filename.ModTime()
	fmt.Println("filetime : ", filetime)

	m := gelf.Message{
		Version: "1.1",
		Short:   msg,
		Full:    messagelog,
		Host:    host,
		Level:   int32(level),
		//MessageLog: messagelog,
		//File:       file,
		//Localtime:  t.Format("01-02-2006T15-04-05"),
		TimeUnix: float64(time.Now().Unix()),
		Facility: "GELF",
		Extra: map[string]interface{}{
			"_file":        file,
			"_environment": environment,
			"_filetime":    filetime,
			"_application": "fred",
		},
	}

	return m
}

func PushToGraylogUdp(dir_watcher Dir_watcher, ip string, payload *gelf.Message) {

	file := payload.Full
	filepath := dir_watcher.Directory + "/" + file

	log.Println("file pathpush :", filepath)

	// get last modified time
	filename, err := os.Stat(filepath)
	if err != nil {
		fmt.Println("error can not stat file : %s", err)
		return
	}

	if size := filename.Size(); size != 0 {
		log.Println("error : file size > 0 : %d", size)
		return
	}

	filetime := filename.ModTime()
	fmt.Println("filetime2 : ", filetime)

	tnow := time.Now()

	// get the diff
	diff := tnow.Sub(filetime)
	fmt.Println("Lifespan is : ", diff)

	duration, _ := time.ParseDuration(dir_watcher.Removetime)
	fmt.Println("Duration : ", duration)

	if diff > duration {
		fmt.Println("> "+dir_watcher.Removetime+" REMOVE : ", filepath)
		var err = os.Remove(filepath)
		if err != nil {
			log.Println("error on delete file %s ,error : %s", filepath, err.Error())
			return
		}

		if len(ip) != 0 {
			gelfWriter, err := gelf.NewTCPWriter(ip)
			if err != nil {
				log.Println("gelf.NewWriter error : %s", err)
			}
			if err := gelfWriter.WriteMessage(payload); err != nil {
				log.Println("gelf.WriteMessage error: %s", err)
			}
			fmt.Println("IP:>", ip)
			fmt.Println("payload: ", payload)
		}

	} else {
		fmt.Println("< ", dir_watcher.Removetime)
	}

	//payload_id := payload.ID
	/*
		if file, ok := payload.Extra["_file"].(string); ok {
			fmt.Println("file error, ", ok)
		} else {

		file_name := filepath.Base(file)
		log.Println("file name :", file_name)
	*/

	//url := "http://192.168.51.57:12201/gelf"
	//	fmt.Println("URL:>", url)
	//	fmt.Println("PAYLOAD ID:>", payload_id)

	//jsonStr, _ := json.Marshal(payload)

	//fmt.Println("json: ", string(jsonStr))

	//graylogAddr = ip:port

	// log to both stderr and graylog2
	//log.SetOutput(io.MultiWriter(os.Stderr, gelfWriter))
	//gelfWriter.Write([]byte(jsonStr))
	//var err error
	/*
		if err := gelfWriter.WriteMessage(payload); err !=nil {
			log.Fatalf("gelf.NewWriter: %s", err)
		}
	*/

	//log.Printf(string(jsonStr))

	//t := time.Now()
	//file_timestamp := t.Format("01-02-2006T15-04-05") + "-" + strconv.Itoa(t.Nanosecond())

	// compare file_timestamp > file.timestamp
	//os.Remove(file)

}

func PushToGraylogHttp(dir_watcher Dir_watcher, ip string, payload *Message) {

	payload_id := payload.ID
	file := payload.File

	file_name := filepath.Base(file)
	log.Println("file name :", file_name)

	//url := "http://192.168.51.57:12201/gelf"
	fmt.Println("IP:>", ip)
	fmt.Println("PAYLOAD ID:>", payload_id)

	jsonStr, _ := json.Marshal(payload)

	fmt.Println("json: ", string(jsonStr))

	req, err := http.NewRequest("POST", ip, bytes.NewBuffer(jsonStr))
	//    req.Header.Set("X-Custom-Header", "myvalue")
	//    req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		//panic(err)
		//log.Fatal("err")
		log.Println("graylog post err  :", err)
		//if watcher_type == "event" {
		//	boltdb.writerChan <- [3]interface{}{"graylog", "", payload}
		//}
		return
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

	if strings.Contains(resp.Status, "202") == true {
		log.Println("status: ", resp.Status)

		//t := time.Now()
		//file_timestamp := t.Format("01-02-2006T15-04-05") + "-" + strconv.Itoa(t.Nanosecond())

		// compare file_timestamp > file.timestamp
		//os.Remove(file)

	} else {
		log.Fatal("Graylog server error : ", resp.Status)
		// store payload to boltdb to send it later
		//storeDB(dbfile, &m)

		//if watcher_type == "event" {
		//	boltdb.writerChan <- [3]interface{}{"graylog", "", payload}
		//}
	}
}

func LogNewWatcher(graylog *Graylog, dir_watcher Dir_watcher) {

	fmt.Println("watched dir :", dir_watcher.Directory)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("created file:", event.Name)
					file_ext := filepath.Ext(event.Name)

					if file_ext == dir_watcher.Ext_file {
						data := event.Name
						log.Println("data: ", string(data))

						//var sem = make(chan int, MaxTaches)
						ip := graylog.Ip + ":" + strconv.Itoa(graylog.Port)
						// fmt.Println("URL : " + url)

						payload := payload(dir_watcher.Environment, dir_watcher.Name, string(data), event.Name, dir_watcher.Payload_host, dir_watcher.Payload_level)

						go PushToGraylogUdp(
							dir_watcher,
							ip,
							&payload)
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(dir_watcher.Directory)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
