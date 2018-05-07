package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pulento/yeelight"
)

var (
	timeSearch = 3
	lights     map[string]*yeelight.Light
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
	log.Printf("Initial lights search for %d [sec]", timeSearch)

	lights, err = yeelight.Search(timeSearch, "")
	if err != nil {
		log.Fatal("Error searching lights cannot continue:", err)
	}
	resnot := make(chan *yeelight.ResultNotification)
	done := make(chan bool)
	for i, l := range lights {
		_, err = l.Listen(resnot)
		if err != nil {
			log.Printf("Error connecting to %s: %s", l.Address, err)
		} else {
			log.Printf("Light %s named %s connected to %s", i, l.Name, l.Address)
		}
	}

	go func(c <-chan *yeelight.ResultNotification, done <-chan bool) {
		log.Println("Channel receiver started")
		for {
			select {
			case <-c:
				{
					// By now just log messages since light data is automatically updated
					data := <-c
					if data != nil {
						if data.Notification != nil {
							log.Println("Notification from Channel", *data.Notification)
						} else {
							log.Println("Result from Channel", *data.Result)
						}
					}
				}
			case <-done:
				return
			}
		}
	}(resnot, done)

	router := mux.NewRouter()
	router.HandleFunc("/", Index).Methods("GET")
	router.HandleFunc("/lights", GetLights).Methods("GET")
	router.HandleFunc("/light/{id}", GetLight).Methods("GET")
	router.HandleFunc("/light/{id}/toggle", ToggleLight).Methods("GET")
	router.HandleFunc("/light/{id}/{command}/{value}", CommandLight).Methods("GET")
	log.Fatal(http.ListenAndServe(":8000", router))
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
	//log.Println(lights)
}

// GetLight returns a light data
func GetLight(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	json.NewEncoder(w).Encode(lights[params["id"]])
	//log.Println(lights[params["id"]])
}

// ToggleLight toggles light power
func ToggleLight(w http.ResponseWriter, r *http.Request) {
	var res APIResult
	params := mux.Vars(r)
	if lights[params["id"]] != nil {
		reqid, err := lights[params["id"]].Toggle()
		r := lights[params["id"]].WaitResult(reqid, 2)
		log.Println("Result received:", *r)
		if err != nil {
			log.Println("Error toggling light:", err)
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
						Params: []string{"error setting brightness"},
					}
				}
				r := l.WaitResult(reqid, 2)
				if r != nil {
					if r.Error != nil {
						log.Println("Error received:", *r.Error)
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
						log.Println("Result received:", *r)
					}
				} else {
					log.Println("Timeout waiting for reply:", reqid)
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
	json.NewEncoder(w).Encode(res)
}
