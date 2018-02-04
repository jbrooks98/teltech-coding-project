package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

type equation struct {
	Action      string  `json:"action"`
	Cached      bool    `json:"cached"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Answer      float64 `json:"answer"`
	redisClient *redis.Client
}

func toFloat(s string) (float64, error) {
	var r float64
	r, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return r, nil
	}
	return r, nil
}

func sendJsonResponse(w http.ResponseWriter, e *equation) error {
	data, err := json.Marshal(e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)

	return nil
}

func getCache(key string, r *redis.Client) (string, error) {
	var v string
	v, err := r.Get(key).Result()
	if err == redis.Nil {
		return v, errors.New("key does not exists")
	}
	// reset the cache
	r.Del(key)
	setCache(r, key, v, time.Minute)

	return v, nil
}

func setCache(r *redis.Client, k, v string, t time.Duration) error {
	err := r.Set(k, v, t).Err()
	if err != nil {
		return err
	}
	return nil
}

func (m equation) multiply() float64 {
	return m.X * m.Y
}

func (d equation) divide() float64 {
	return d.X / d.Y
}

func (a equation) add() float64 {
	return a.X + a.Y
}

func (s equation) subtract() float64 {
	return s.X - s.Y
}

func (e *equation) equationHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var v string
	e.Action = r.URL.Path[len("/"):]

	if err = r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	e.X, err = toFloat(r.Form.Get("x"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	e.Y, err = toFloat(r.Form.Get("y"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	key := fmt.Sprintf("%g%s%g", e.X, e.Action, e.Y)
	value, err := getCache(key, e.redisClient)
	if err != nil {
		e.Cached = false
	} else {
		e.Cached = true
		e.Answer, err = toFloat(value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		sendJsonResponse(w, e)
		return
	}
	switch {
	case e.Action == "multiply":
		e.Answer = e.multiply()
		v = fmt.Sprintf("%g", e.Answer)
	case e.Action == "divide":
		if e.X == 0 {
			http.Error(w, errors.New("cannot divide by zero").Error(), http.StatusBadRequest)
			return
		}
		e.Answer = e.divide()
		v = fmt.Sprintf("%g", e.Answer)
	case e.Action == "add":
		e.Answer = e.add()
		v = fmt.Sprintf("%g", e.Answer)
	case e.Action == "subtract":
		e.Answer = e.subtract()
		v = fmt.Sprintf("%g", e.Answer)
	default:
		http.Error(w, "Invalid operator", http.StatusBadRequest)
		return
	}
	setCache(e.redisClient, key, v, time.Minute)
	sendJsonResponse(w, e)
}

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	_, err := client.Ping().Result()
	if err != nil {
		log.Fatalf(err.Error())
	}
	e := &equation{"", false, 0, 0, 0, client}
	http.HandleFunc("/", e.equationHandler)

	log.Fatalf("Server failed to listen: %s", http.ListenAndServe(":8081", nil))
}
