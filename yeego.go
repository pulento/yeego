package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/pulento/yeelight"
)

var (
	timeSearch     = 3
	lights         = make(map[string]*yeelight.Light)
	commandTimeout = 2
	ssdpRescan     = 3 * time.Minute
)

// APIResult is the response to a command
type APIResult struct {
	Result string          `json:"result"`
	ID     string          `json:"id,omitempty"`
	Params []string        `json:"params,omitempty"`
	Error  *yeelight.Error `json:"error,omitempty"`
}

// Let's roll
func main() {
	var err error
	//defer profile.Start(profile.MemProfile).Stop()
	//defer profile.Start().Stop()

	log.SetLevel(log.DebugLevel)
	log.Printf("Initial lights search for %d [sec]", timeSearch)

	// Start a result/notification listener for each light
	resnot := make(chan *yeelight.ResultNotification)
	done := make(chan bool)

	// Initial search
	err = yeelight.Search(timeSearch, "", lights, func(l *yeelight.Light) {
		_, lerr := l.Listen(resnot)
		if lerr != nil {
			log.Errorf("Error connecting to %s: %s", l.Address, err)
		}
	})
	if err != nil {
		log.Fatal("Error searching lights cannot continue:", err)
	}
	/*
		for _, l := range lights {
			_, err = l.Listen(resnot)
			if err != nil {
				log.Errorf("Error connecting to %s: %s", l.Address, err)
			}
		}*/
	log.Printf("Found %d lights", len(lights))

	// Start a SSDP monitor for lights traffic
	err = yeelight.SSDPMonitor(lights, func(l *yeelight.Light) {
		if l.Conn == nil {
			// A new light is automatically added but not connected
			l.Listen(resnot)
		}
	})

	if err != nil {
		log.Errorln("Error starting SSDP monitor", err)
	}

	go func(c <-chan *yeelight.ResultNotification, done <-chan bool) {
		log.Debug("Messages receiver started")
		for {
			select {
			case data := <-c:
				// By now just log messages since light data is automatically updated
				if data != nil {
					if data.Notification != nil {
						log.WithFields(log.Fields{
							"ID":     (*data.Notification).DevID,
							"method": (*data.Notification).Method,
						}).Debugln("Notification:", (*data.Notification).Params)
					} else {
						log.WithFields(log.Fields{
							"mID": (*data.Result).ID,
							"ID":  (*data.Result).DevID,
						}).Debugln("Result: ", (*data.Result).Result)
					}
				}
			case <-done:
				return
			}
		}
	}(resnot, done)

	// Every 3 min run a SSDP search again
	go func(refresh <-chan time.Time) {
		select {
		case <-refresh:
			log.Info("SSDP rescan")
			refresh = time.After(ssdpRescan)
			err = yeelight.Search(timeSearch, "", lights, func(l *yeelight.Light) {
				_, lerr := l.Listen(resnot)
				if lerr != nil {
					log.Errorf("Error connecting to %s: %s", l.Address, err)
				}
			})
			if err != nil {
				log.Errorln("Error on SSDP rescan: ", err)
			}
		}
	}(time.After(ssdpRescan))

	var dir = "views"

	// Serving HTTP port
	port := os.Getenv("YEEGO_PORT")
	if port == "" {
		port = "8080"
	}

	router := mux.NewRouter()

	router.HandleFunc("/light", GetLights).Methods("GET")
	router.HandleFunc("/lights", GetLights).Methods("GET")
	router.HandleFunc("/light/{id}", GetLight).Methods("GET")
	router.HandleFunc("/light/{id}/toggle", ToggleLight).Methods("GET")
	router.HandleFunc("/light/{id}/{command}/{value}", CommandLight).Methods("GET")

	// Profile stuff
	router.HandleFunc("/_count", GetGoroutinesCount).Methods("GET")
	router.HandleFunc("/_stack", GetStackTrace).Methods("GET")

	// This will serve files static content
	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(dir))))

	log.Info("Listening HTTP: ", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

// GetGoroutinesCount returns Goroutines count
func GetGoroutinesCount(w http.ResponseWriter, r *http.Request) {
	// Get the count of number of go routines running.
	count := runtime.NumGoroutine()
	json.NewEncoder(w).Encode(count)
	//w.Write([]byte(strconv.Itoa(count)))
}

// GetStackTrace dumps stack trace
func GetStackTrace(w http.ResponseWriter, r *http.Request) {
	stack := debug.Stack()
	w.Write(stack)
	pprof.Lookup("goroutine").WriteTo(w, 2)
}

// Index does nothing
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Yeego")
}

// GetLights returns all lights data
func GetLights(w http.ResponseWriter, r *http.Request) {
	var ls []*yeelight.Light
	for _, light := range lights {
		ls = append(ls, light)
	}
	json.NewEncoder(w).Encode(ls)
}

// GetLight returns a light data
func GetLight(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	json.NewEncoder(w).Encode(lights[params["id"]])
}

// ToggleLight toggles light power
func ToggleLight(w http.ResponseWriter, r *http.Request) {
	var res APIResult
	params := mux.Vars(r)
	if lights[params["id"]] != nil {
		reqid, err := lights[params["id"]].Toggle()
		lights[params["id"]].WaitResult(reqid, commandTimeout)
		if err != nil {
			log.Errorln("Error toggling light:", err)
		} else {
			res = APIResult{
				Result: "ok",
				ID:     lights[params["id"]].ID,
			}
		}
	} else {
		res = APIResult{
			Result: "not found",
		}
	}
	json.NewEncoder(w).Encode(res)
}

// CommandLight sends a command with its parameter
func CommandLight(w http.ResponseWriter, r *http.Request) {
	var res APIResult
	var err error
	p := mux.Vars(r)

	l := lights[p["id"]]
	if l != nil {
		var value int
		if p["value"] != "" {
			value, err = strconv.Atoi(p["value"])
			if err != nil {
				res = APIResult{
					Result: "error",
					Params: []string{"invalid value"},
				}
			}
		}
		if p["command"] == "brightness" {
			if err == nil {
				var reqid int32
				reqid, err = l.SetBrightness(value, 0)
				if err != nil {
					res = APIResult{
						Result: "error",
						Params: []string{err.Error()},
					}
					log.Errorln("Error setting brightness:", err)
					goto end
				}
				r := l.WaitResult(reqid, commandTimeout)
				if r != nil {
					if r.Error != nil {
						log.Errorln("Error received:", *r.Error)
						res = APIResult{
							Result: "error",
							ID:     l.ID,
							Error:  r.Error,
						}
					} else {
						res = APIResult{
							Result: "ok",
							ID:     l.ID,
						}
					}
				} else {
					log.Warnln("Timeout waiting for reply:", reqid)
					res = APIResult{
						Result: "error",
						Params: []string{"timeout setting brightness"},
					}
				}
			} else {
				res = APIResult{
					Result: "error",
					Params: []string{"invalid value"},
				}
			}
		} else if p["command"] == "setname" {
			var reqid int32
			name := p["value"]

			reqid, err = l.SetName(name, 0)
			if err != nil {
				res = APIResult{
					Result: "error",
					Params: []string{err.Error()},
				}
				log.Errorln("Error setting name:", err)
				goto end
			}
			l.WaitResult(reqid, commandTimeout)
		} else {
			res = APIResult{
				Result: "error",
				Params: []string{"invalid command"},
			}
		}
	} else {
		res = APIResult{
			Result: "error",
			Params: []string{"not found"},
		}
	}
end:
	json.NewEncoder(w).Encode(res)
}
