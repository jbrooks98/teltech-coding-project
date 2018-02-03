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

func sendJsonResponse(w http.ResponseWriter, responseBody *equation) error {
	data, err := json.Marshal(responseBody)
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

func (m *equation) multiply() error {
	key := fmt.Sprintf("%d * %d", m.X, m.Y)
	v, err := getCache(key, m.redisClient)
	if err != nil {
		m.Answer = m.X * m.Y
		a := strconv.FormatFloat(m.Answer, 'f', 2, 64)
		setCache(m.redisClient, key, a, time.Minute)
	} else {
		m.Cached = true
		m.Answer, err = toFloat(v)
		if err != nil {
			return errors.New("error converting so integer")
		}
	}
	return nil
}

func (d *equation) divide() error {
	if d.X == 0 {
		return errors.New("cannot divide by zero")
	}
	key := fmt.Sprintf("%d / %d", d.X, d.Y)
	v, err := getCache(key, d.redisClient)
	if err != nil {
		d.Answer = d.X / d.Y
		d.Cached = false
		a := strconv.FormatFloat(d.Answer, 'f', 6, 64)
		setCache(d.redisClient, key, a, time.Minute)
	} else {
		d.Cached = true
		d.Answer, err = toFloat(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *equation) add() error {
	key := fmt.Sprintf("%d + %d", a.X, a.Y)
	v, err := getCache(key, a.redisClient)
	if err != nil {
		a.Answer = a.X + a.Y
		s := strconv.FormatFloat(a.Answer, 'f', 6, 64)
		setCache(a.redisClient, key, s, time.Minute)
	} else {
		a.Cached = true
		a.Answer, err = toFloat(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *equation) subtract() error {
	key := fmt.Sprintf("%d - %d", s.X, s.Y)
	v, err := getCache(key, s.redisClient)
	if err != nil {
		s.Answer = s.X - s.Y
		a := strconv.FormatFloat(s.Answer, 'f', 6, 64)
		setCache(s.redisClient, key, a, time.Minute)
	} else {
		s.Cached = true
		s.Answer, err = toFloat(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *equation) equationHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	e.Action = r.URL.Path[len("/"):]

	if err = r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	e.X, err = toFloat(r.Form.Get("x"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	e.Y, err = toFloat(r.Form.Get("y"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	switch {
	case e.Action == "multiply":
		err = e.multiply()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		sendJsonResponse(w, e)
	case e.Action == "divide":
		err = e.divide()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		sendJsonResponse(w, e)
	case e.Action == "add":
		err = e.add()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		sendJsonResponse(w, e)
	case e.Action == "subtract":
		err = e.subtract()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		sendJsonResponse(w, e)
	default:
		http.Error(w, "Invalid operator", http.StatusBadRequest)
	}
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
