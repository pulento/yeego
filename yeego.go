package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"bitbucket.org/pulento/yeelight"
	"github.com/gorilla/mux"
)

var (
	timeSearch = 3
	lights     map[string]*yeelight.Light
)

// Let's roll
func main() {
	var err error

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
					data := <-c
					if data.Notification != nil {
						log.Println("Notification from Channel", *data.Notification)
					} else {
						log.Println("Result from Channel", *data.Result)
					}
				}
			case <-done:
				return
			}
		}
	}(resnot, done)

	for _, l := range lights {
		prop := "power"
		err := l.GetProp(prop, "bright")
		if err != nil {
			log.Printf("Error getting property %s on %s: %s", prop, l.Address, err)
		}
	}

	router := mux.NewRouter()
	router.HandleFunc("/", Index).Methods("GET")
	router.HandleFunc("/lights", GetLights).Methods("GET")
	router.HandleFunc("/light/{id}", GetLight).Methods("GET")
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
	log.Println(lights)
}

// GetLight returns a light data
func GetLight(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	json.NewEncoder(w).Encode(lights[params["id"]])
	log.Println(lights[params["id"]])
}
