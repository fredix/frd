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

type Watcher struct {
	Watcher_type  string
	Name          string
	Environment   string
	Directory     string
	Ext_file      string
	File_size     string
	Size_unit     string
	Payload_host  string
	Payload_level int
	Removetime    string
	Loop_sleep    string
}

func LoopDirectory(graylog *Graylog, watcher Watcher) {
	go func() {
		for {
			ListFilesAndPush(graylog, watcher)
			//time.Sleep(60000 * time.Millisecond)
			fmt.Println("watcher.Loop_sleep : ", watcher.Loop_sleep)
			fmt.Println("watcher.Ext_file : ", watcher.Ext_file)
			duration, _ := time.ParseDuration(watcher.Loop_sleep)
			fmt.Println("Duration sleep : ", duration)
			time.Sleep(duration)
		}
	}()
}

func ListFilesAndPush(graylog *Graylog, watcher Watcher) {
	var ip string = ""
	if len(graylog.Ip) != 0 && graylog.Port != 0 {
		ip = graylog.Ip + ":" + strconv.Itoa(graylog.Port)
	}

	if len(watcher.Ext_file) != 0 {
		files, _ := ioutil.ReadDir(watcher.Directory)
		for _, f := range files {
			file_ext := filepath.Ext(f.Name())
			if file_ext == watcher.Ext_file || watcher.Ext_file == "*" {
				fmt.Println("file to remove : ", watcher.Directory+"/"+f.Name())
				payload := payload(watcher.Environment, watcher.Name, f.Name(), watcher.Directory+"/"+f.Name(), watcher.Payload_host, watcher.Payload_level)
				RemoveFile(
					watcher,
					ip,
					&payload)
			}
		}
	} else {
		return
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

	filesize := filename.Size()

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
			"_size":        filesize,
			"_environment": environment,
			"_filetime":    filetime,
			"_application": "fred",
		},
	}

	return m
}

func RemoveFile(watcher Watcher, ip string, payload *gelf.Message) {
	file := payload.Extra["_file"].(string)
	log.Println("file to remove :", file)

	// get last modified time
	filename, err := os.Stat(file)
	if err != nil {
		fmt.Println("error can not stat file : %s", err)
		return
	}

	filesize_bytes := filename.Size()
	fmt.Println("filesize_bytes : %s", filesize_bytes)

	lspecifiedsize := watcher.File_size[1:len(watcher.File_size)]
	fmt.Println("Lspecifiedsize : %s", lspecifiedsize)

	specifiedsize, err := strconv.Atoi(watcher.File_size[1:len(watcher.File_size)])
	if err != nil {
		fmt.Println("unable to extract file size watcher.File_size : ", watcher.File_size)
		return
	}

	fmt.Println("specifiedsize : %s", specifiedsize)

	// convert specifiedsize to bytes
	var filesize_kilobytes int64 = 0
	var filesize_float64 float64 = 0

	switch watcher.Size_unit {
	case "bytes":
		filesize_kilobytes = filesize_bytes

	case "kilobytes":
		filesize_kilobytes = (filesize_bytes / 1024)

	case "megabytes":
		filesize_kilobytes = (filesize_bytes / 1024)
		filesize_megabytes := (float64)(filesize_kilobytes / 1024) // cast to type float64
		filesize_float64 = filesize_megabytes
	case "gigabytes":
		filesize_kilobytes = (filesize_bytes / 1024)
		filesize_megabytes := (float64)(filesize_kilobytes / 1024) // cast to type float64
		filesize_gigabytes := (filesize_megabytes / 1024)
		filesize_float64 = filesize_gigabytes

	case "terabytes":
		filesize_kilobytes = (filesize_bytes / 1024)
		filesize_megabytes := (float64)(filesize_kilobytes / 1024) // cast to type float64
		filesize_gigabytes := (filesize_megabytes / 1024)
		filesize_terabytes := (filesize_gigabytes / 1024)
		filesize_float64 = filesize_terabytes

	default:
		fmt.Println("size_unit failed : ", watcher.Size_unit)
		return
	}

	switch operator := watcher.File_size[0:1]; operator {
	case "=":
		if filesize_float64 != 0 {
			if int(filesize_float64) != specifiedsize {
				fmt.Println("filesize not equal  : %f, %d", filesize_float64, specifiedsize)
				return
			}
		} else if int(filesize_kilobytes) != specifiedsize {
			fmt.Println("filesize not equal  : %d, %d", filesize_kilobytes, specifiedsize)
			return
		}

	case "<":
		if filesize_float64 != 0 {
			if int(filesize_float64) >= specifiedsize {
				fmt.Println("filesize not <  : %f, %d", filesize_float64, specifiedsize)
				return
			}
		} else if int(filesize_kilobytes) >= specifiedsize {
			fmt.Println("filesize not <  : %d, %d", filesize_kilobytes, specifiedsize)
			return
		}

	case ">":
		if filesize_float64 != 0 {
			if int(filesize_float64) <= specifiedsize {
				fmt.Println("filesize not >  : %f, %d", filesize_float64, specifiedsize)
				return
			}
		} else if int(filesize_kilobytes) <= specifiedsize {
			fmt.Println("filesize not >  : %d, %d", filesize_kilobytes, specifiedsize)
			return
		}

		// fmt.Println("file_size operator : %s, %s, %s", operator, filesize_kilobytes, specifiedsize)
	default:
		fmt.Println("file_size operator error : %s", operator)
		return
	}

	// sized tests are passed

	filetime := filename.ModTime()
	fmt.Println("filetime : ", filetime)

	tnow := time.Now()

	// get the diff
	diff := tnow.Sub(filetime)
	fmt.Println("Lifespan is : ", diff)

	duration, _ := time.ParseDuration(watcher.Removetime)
	fmt.Println("Duration : ", duration)

	if diff > duration {
		fmt.Println("> "+watcher.Removetime+" REMOVE : ", file)
		var err = os.Remove(file)
		if err != nil {
			log.Println("error on delete file %s ,error : %s", file, err.Error())
			return
		}

		if len(ip) != 0 {
			PushToGraylogUdp(ip, payload)
		}

	} else {
		fmt.Println("< ", watcher.Removetime)
	}
}

func PushToGraylogUdp(ip string, payload *gelf.Message) {
	gelfWriter, err := gelf.NewUDPWriter(ip)
	if err != nil {
		log.Println("gelf.NewUDPWriter error : %s", err)
		return
	}
	if err := gelfWriter.WriteMessage(payload); err != nil {
		log.Println("gelf.WriteMessage error: %s", err)
		return
	}
	fmt.Println("IP:>", ip)
	fmt.Println("payload: ", payload)
}

func PushToGraylogHttp(watcher Watcher, ip string, payload *Message) {

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

func LogNewWatcher(graylog *Graylog, watcher Watcher) {

	fmt.Println("watched dir :", watcher.Directory)

	new_watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer new_watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-new_watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("created file:", event.Name)
					file_ext := filepath.Ext(event.Name)

					if file_ext == watcher.Ext_file || watcher.Ext_file == "*" {
						data := event.Name
						log.Println("event.Name: ", string(data))

						//var sem = make(chan int, MaxTaches)
						var ip string = ""
						if len(graylog.Ip) != 0 && graylog.Port != 0 {
							ip = graylog.Ip + ":" + strconv.Itoa(graylog.Port)
						}
						payload := payload(watcher.Environment, watcher.Name, string(data), event.Name, watcher.Payload_host, watcher.Payload_level)

						go RemoveFile(
							watcher,
							ip,
							&payload)
					}
				}
			case err := <-new_watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = new_watcher.Add(watcher.Directory)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
