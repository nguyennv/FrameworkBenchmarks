package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"sort"
	"strconv"
)

const (
	connectionString = "localhost"
	helloWorldString = "Hello, world!"
	worldRowCount    = 10000
)

var (
	tmpl = template.Must(template.ParseFiles("templates/layout.html", "templates/fortune.html"))

	database *mgo.Database
	fortunes *mgo.Collection
	worlds   *mgo.Collection
)

type Message struct {
	Message string `json:"message"`
}

type World struct {
	Id           uint16 `json:"id"`
	RandomNumber uint16 `json:"randomNumber"`
}

type Fortune struct {
	Id      uint16 `json:"id"`
	Message string `json:"message"`
}

type Fortunes []Fortune

func (s Fortunes) Len() int {
	return len(s)
}

func (s Fortunes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type ByMessage struct{ Fortunes }

func (s ByMessage) Less(i, j int) bool {
	return s.Fortunes[i].Message < s.Fortunes[j].Message
}

func main() {
	port := ":8228"
	runtime.GOMAXPROCS(runtime.NumCPU())
	if session, err := mgo.Dial(connectionString); err != nil {
		log.Fatalf("Error opening database: %v", err)
	} else {
		defer session.Close()
		session.SetPoolLimit(5)
		database = session.DB("hello_world")
		worlds = database.C("world")
		fortunes = database.C("fortune")
		http.HandleFunc("/json", jsonHandler)
		http.HandleFunc("/db", dbHandler)
		http.HandleFunc("/fortune", fortuneHandler)
		http.HandleFunc("queries", queriesHandler)
		http.HandleFunc("/update", updateHandler)
		http.HandleFunc("/plaintext", plaintextHandler)
		fmt.Println("Serving on http://localhost" + port)
		http.ListenAndServe(port, nil)
	}
}

// Helper for random numbers
func getRandomNumber() int {
	return rand.Intn(worldRowCount) + 1
}

// Test 1: JSON serialization
func jsonHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	json.NewEncoder(w).Encode(&Message{helloWorldString})
}

func dbHandler(w http.ResponseWriter, r *http.Request) {
	var world World
	query := bson.M{"id": getRandomNumber()}
	if worlds != nil {
		if err := worlds.Find(query).One(&world); err != nil {
			log.Fatalf("Error finding world with id: %s", err.Error())
			return
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&world)
			return
		}
	} else {
		log.Fatal("Collection not initialized properly")
	}
}

func queriesHandler(w http.ResponseWriter, r *http.Request) {
	n := 1
	if nStr := r.URL.Query().Get("queries"); len(nStr) > 0 {
		n, _ = strconv.Atoi(nStr)
	}

	if n <= 1 {
		dbHandler(w, r)
		return
	} else if n > 500 {
		n = 500
	}

	result := make([]World, n)
	for _, world := range result {
		query := bson.M{"id": getRandomNumber()}
		if err := worlds.Find(query).One(&world); err != nil {
			log.Fatalf("Error finding world with id: %s", err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func fortuneHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	f := make(Fortunes, 16)
	if err := fortunes.Find(nil).All(&f); err == nil {
		f = append(f, Fortune{
			Message: "Additional fortune added at request time.",
		})
		sort.Sort(ByMessage{f})
		if err := tmpl.Execute(w, f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	n := 1
	if nStr := r.URL.Query().Get("queries"); len(nStr) > 0 {
		n, _ = strconv.Atoi(nStr)
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	if n <= 1 {
		var world World
		colQuery := bson.M{"id": getRandomNumber()}
		update := bson.M{"$set": bson.M{"randomNumber": getRandomNumber()}}
		if err := worlds.Update(colQuery, update); err != nil {
			log.Fatalf("Error updating world with id: %s", err.Error())
		} else {
			world.Id = colQuery["id"].(uint16)
			world.RandomNumber = update["$set"].(bson.M)["randomNumber"].(uint16)
		}
		encoder.Encode(world)
	} else {
		if n > 500 {
			n = 500
		}
		result := make([]World, n)
		for _, world := range result {
			colQuery := bson.M{"id": getRandomNumber()}
			update := bson.M{"$set": bson.M{"randomNumber": getRandomNumber()}}
			if err := worlds.Update(colQuery, update); err != nil {
				log.Fatalf("Error updating world with id: %s", err.Error())
			} else {
				world.Id = colQuery["id"].(uint16)
				world.RandomNumber = update["$set"].(bson.M)["randomNumber"].(uint16)
			}
		}
		encoder.Encode(result)
	}
}

func plaintextHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(helloWorldString))
}
