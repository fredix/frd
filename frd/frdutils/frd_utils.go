package frdutils

import (
	"github.com/fredix/frd/frd/frdlog"

	// standard packages
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	// external packages
	//"github.com/Graylog2/go-gelf/gelf"
	"github.com/fsnotify/fsnotify"
	//"gopkg.in/Graylog2/go-gelf.v2/gelf"
	"gopkg.in/Graylog2/go-gelf.v1/gelf"
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
	Watcher_type     string
	Name             string
	Environment      string
	Directory        string
	Recursive        bool
	External_rm      string
	External_options string
	Ext_file         string
	File_size        string
	Size_unit        string
	Payload_host     string
	Payload_level    int
	Removetime       string
	Loop_sleep       string
}

var counter = struct {
	sync.RWMutex
	map_files map[string]bool
}{map_files: make(map[string]bool)}

func LoopDirectory(frdconf *frdlog.FrdConfig, graylog *Graylog, watcher Watcher) {
	go func() {
		for {
			ListFilesAndRemove(graylog, frdconf, watcher)
			//time.Sleep(60000 * time.Millisecond)
			frdlog.PrintLog(frdconf, "watcher.Loop_sleep : "+watcher.Loop_sleep)
			frdlog.PrintLog(frdconf, "watcher.Ext_file : "+watcher.Ext_file)
			duration, _ := time.ParseDuration(watcher.Loop_sleep)
			frdlog.PrintLog(frdconf, "Duration sleep : "+duration.String())
			time.Sleep(duration)
		}
	}()
}

func ListFilesAndRemove(graylog *Graylog, frdconf *frdlog.FrdConfig, watcher Watcher) {

	var ip string = ""
	if len(graylog.Ip) != 0 && graylog.Port != 0 {
		ip = graylog.Ip + ":" + strconv.Itoa(graylog.Port)
	}

	if len(watcher.Ext_file) != 0 {
		files, _ := ioutil.ReadDir(watcher.Directory)
		for _, f := range files {
			file_ext := filepath.Ext(f.Name())
			if file_ext == watcher.Ext_file || watcher.Ext_file == "*" {
				//fmt.Println("ListFilesAndRemove file to remove : ", watcher.Directory+"/"+f.Name())
				payload, err := payload(watcher.Environment, watcher.Name, f.Name(), watcher.Directory+"/"+f.Name(), watcher.Payload_host, watcher.Payload_level)
				if err == nil {
					RemoveFile(
						frdconf,
						watcher,
						ip,
						&payload)
				} else {
					frdlog.PrintLog(frdconf, "payload error : "+err.Error())
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
	} else {
		return
	}
}

func payload(environment string, msg string, messagelog string, file string, host string, level int) (gelf.Message, error) {

	// get last modified time
	filename, err := os.Stat(file)
	if err != nil {
		return gelf.Message{}, errors.New("can not stat file : " + err.Error())
	}

	counter.RLock()
	if _, ok := counter.map_files[file]; ok {
		//fmt.Println("File exist :", ok)
		counter.RUnlock()
		return gelf.Message{}, errors.New("file already exist in map")
	}
	counter.RUnlock()

	filetime := filename.ModTime()
	//fmt.Println("filetime : ", filetime)

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
			"_application": "frd",
		},
	}

	return m, nil
}

func RemoveFile(frdconf *frdlog.FrdConfig, watcher Watcher, ip string, payload *gelf.Message) {
	file := payload.Extra["_file"].(string)
	//log.Println("check file to remove :", file)

	// get last modified time
	filename, err := os.Stat(file)
	if err != nil {
		frdlog.PrintLog(frdconf, "RemoveFile error can not stat file : "+err.Error())
		return
	}

	if filename.IsDir() == false {
		//fmt.Println("not directory : ", filename)
		filesize_bytes := filename.Size()
		//fmt.Println("filesize_bytes : %s", filesize_bytes)

		specifiedsize, err := strconv.Atoi(watcher.File_size[1:len(watcher.File_size)])
		if err != nil {
			frdlog.PrintLog(frdconf, "unable to extract file size watcher.File_size : "+watcher.File_size)
			return
		}

		//fmt.Println("specifiedsize : %s", specifiedsize)

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
			frdlog.PrintLog(frdconf, "size_unit failed : "+watcher.Size_unit)
			return
		}

		switch operator := watcher.File_size[0:1]; operator {
		case "=":
			if filesize_float64 != 0 {
				if int(filesize_float64) != specifiedsize {
					frdlog.PrintLog(frdconf, "filesize not equal : "+frdlog.FloatToString(filesize_float64)+" <> "+strconv.Itoa(specifiedsize))
					//fmt.Println("filesize not equal  : %f, %d", filesize_float64, specifiedsize)
					return
				}
			} else if int(filesize_kilobytes) != specifiedsize {
				frdlog.PrintLog(frdconf, "filesize not equal : "+strconv.Itoa(int(filesize_kilobytes))+" <> "+strconv.Itoa(specifiedsize))
				// fmt.Println("filesize not equal  : %d, %d", filesize_kilobytes, specifiedsize)
				return
			}

		case "<":
			if filesize_float64 != 0 {
				if int(filesize_float64) >= specifiedsize {
					frdlog.PrintLog(frdconf, "filesize : "+frdlog.FloatToString(filesize_float64)+" >= "+strconv.Itoa(specifiedsize))
					//fmt.Println("filesize not <  : %f, %d", filesize_float64, specifiedsize)
					return
				}
			} else if int(filesize_kilobytes) >= specifiedsize {
				frdlog.PrintLog(frdconf, "filesize : "+strconv.Itoa(int(filesize_kilobytes))+" > "+strconv.Itoa(specifiedsize))
				//fmt.Println("filesize not <  : %d, %d", filesize_kilobytes, specifiedsize)
				return
			}

		case ">":
			if filesize_float64 != 0 {
				if int(filesize_float64) <= specifiedsize {
					frdlog.PrintLog(frdconf, "filesize : "+frdlog.FloatToString(filesize_float64)+" < "+strconv.Itoa(specifiedsize))
					//fmt.Println("filesize not >  : %f, %d", filesize_float64, specifiedsize)
					return
				}
			} else if int(filesize_kilobytes) <= specifiedsize {
				frdlog.PrintLog(frdconf, "filesize : "+strconv.Itoa(int(filesize_kilobytes))+" < "+strconv.Itoa(specifiedsize))
				//fmt.Println("filesize not >  : %d, %d", filesize_kilobytes, specifiedsize)
				return
			}

			// fmt.Println("file_size operator : %s, %s, %s", operator, filesize_kilobytes, specifiedsize)
		default:
			frdlog.PrintLog(frdconf, "file_size operator error : "+operator)
			return
		}
	}
	// sized tests ok
	// now check gap timestamp

	filetime := filename.ModTime()
	//fmt.Println("filetime : ", filetime)

	tnow := time.Now()

	// get the diff
	diff := tnow.Sub(filetime)
	//fmt.Println("Lifespan is : ", diff)

	duration, _ := time.ParseDuration(watcher.Removetime)
	//fmt.Println("Duration : ", duration)

	if diff > duration {
		frdlog.PrintLog(frdconf, "> "+watcher.Removetime+" REMOVE : "+file)

		if filename.IsDir() {
			if watcher.Recursive == true {
				frdlog.PrintLog(frdconf, "Remove directory : "+filename.Name())
				if len(watcher.External_rm) == 0 {
					err := os.RemoveAll(file)
					if err != nil {
						frdlog.PrintLog(frdconf, "Remove directory error : "+err.Error())
						return
					}
				} else {
					go secureDelete(frdconf, watcher.External_rm, watcher.External_options, file)
				}
			} else {
				frdlog.PrintLog(frdconf, "config file is set to recursive=false (do not remove directory) : "+filename.Name())
				return
			}
		} else {
			if len(watcher.External_rm) == 0 {
				var err = os.Remove(file)
				if err != nil {
					frdlog.PrintLog(frdconf, "error on delete file : "+file+" , error : "+err.Error())
					return
				}
			} else {
				go secureDelete(frdconf, watcher.External_rm, watcher.External_options, file)
			}

		}

		if len(ip) != 0 {
			PushToGraylogUdp(frdconf, ip, payload)
		}
	}
	//	} else {
	//		fmt.Println(filename.Name(), " < ", watcher.Removetime)
	//	}
}

func secureDelete(frdconf *frdlog.FrdConfig, command string, command_options string, file string) {
	cmd := exec.Command(command, command_options, file)
	//cmd := exec.Command(command, command_options)
	outBytes := &bytes.Buffer{}
	errBytes := &bytes.Buffer{}
	cmd.Stdout = outBytes
	cmd.Stderr = errBytes

	frdlog.PrintLog(frdconf, fmt.Sprintf("==> Executing: %s\n", strings.Join(cmd.Args, " ")))

	counter.RLock()
	counter.map_files[file] = true
	counter.RUnlock()

	frdlog.PrintLog(frdconf, "secure delete command : "+command+" "+command_options+" "+file)
	err := cmd.Start()
	if err != nil {
		frdlog.PrintLog(frdconf, "secure delete failed to start : "+err.Error())
		counter.RLock()
		delete(counter.map_files, file)
		counter.RUnlock()
		return
	}

	err = cmd.Wait()
	if err != nil {
		frdlog.PrintLog(frdconf, "secure delete error : "+err.Error())
		counter.RLock()
		delete(counter.map_files, file)
		counter.RUnlock()
		return
	} else {
		frdlog.PrintLog(frdconf, "SECURE DELETED : "+file)
	}

	counter.RLock()
	delete(counter.map_files, file)
	counter.RUnlock()
}

func PushToGraylogUdp(frdconf *frdlog.FrdConfig, ip string, payload *gelf.Message) {
	gelfWriter, err := gelf.NewWriter(ip)
	if err != nil {
		frdlog.PrintLog(frdconf, "gelf.NewWriter error : "+err.Error())
		return
	}
	if err := gelfWriter.WriteMessage(payload); err != nil {
		frdlog.PrintLog(frdconf, "gelf.WriteMessage error: %s"+err.Error())
		return
	}
	frdlog.PrintLog(frdconf, "IP:>"+ip)
	frdlog.PrintLog(frdconf, fmt.Sprintln("payload: ", payload))
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

func LogNewWatcher(frdconf *frdlog.FrdConfig, graylog *Graylog, watcher Watcher) {

	frdlog.PrintLog(frdconf, "watched dir: "+watcher.Directory)

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
				frdlog.PrintLog(frdconf, "event: "+event.String())

				if event.Op&fsnotify.Create == fsnotify.Create {
					frdlog.PrintLog(frdconf, "created file: "+event.Name)
					file_ext := filepath.Ext(event.Name)

					if file_ext == watcher.Ext_file || watcher.Ext_file == "*" {
						data := event.Name
						frdlog.PrintLog(frdconf, "event.Name: "+string(data))

						//var sem = make(chan int, MaxTaches)
						var ip string = ""
						if len(graylog.Ip) != 0 && graylog.Port != 0 {
							ip = graylog.Ip + ":" + strconv.Itoa(graylog.Port)
						}
						payload, err := payload(watcher.Environment, watcher.Name, string(data), event.Name, watcher.Payload_host, watcher.Payload_level)
						if err == nil {
							go RemoveFile(
								frdconf,
								watcher,
								ip,
								&payload)
						}
					}
				}
			case err := <-new_watcher.Errors:
				frdlog.PrintLog(frdconf, "error: "+err.Error())
			}
		}
	}()

	err = new_watcher.Add(watcher.Directory)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
