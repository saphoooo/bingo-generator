package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/trigger", trigger).Methods("GET")
	log.Print("Start listening on :8000...")
	err := http.ListenAndServe(":8000", r)
	if err != nil {
		log.Panic().Msg(err.Error())
	}
}

func trigger(w http.ResponseWriter, r *http.Request) {
	pool := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "redis-master:6379")
		},
	}

	conn := pool.Get()
	_, err := conn.Do("AUTH", os.Getenv("REDIS_PASSWORD"))
	if err != nil {
		log.Error().
			Str("hostname", r.Host).
			Str("method", r.Method).
			Str("proto", r.Proto).
			Str("remote_ip", r.RemoteAddr).
			Str("path", r.RequestURI).
			Str("user-agent", r.UserAgent()).
			Int("status", http.StatusInternalServerError).
			Msg(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Oops something wrong happened...")
		return
	}
	defer conn.Close()

	exists, err := redis.Int(conn.Do("EXISTS", "bingoNumberOfTheDay"))
	if err != nil {
		log.Error().
			Str("hostname", r.Host).
			Str("method", r.Method).
			Str("proto", r.Proto).
			Str("remote_ip", r.RemoteAddr).
			Str("path", r.RequestURI).
			Str("user-agent", r.UserAgent()).
			Int("status", http.StatusInternalServerError).
			Msg(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Oops something wrong happened...")
		return
	}
	if exists == 0 { // the key does not exist
		// create a new key entry
		rand.Seed(time.Now().UTC().UnixNano())
		bingoNumberOfTheDay := rand.Intn(10) + 1
		_, err = conn.Do("SET", "bingoNumberOfTheDay", bingoNumberOfTheDay)
		if err != nil {
			log.Error().
				Str("hostname", r.Host).
				Str("method", r.Method).
				Str("proto", r.Proto).
				Str("remote_ip", r.RemoteAddr).
				Str("path", r.RequestURI).
				Str("user-agent", r.UserAgent()).
				Int("status", http.StatusInternalServerError).
				Msg(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "Oops something wrong happened...")
			return
		}
		// set expiry of 1 day
		_, err = conn.Do("EXPIRE", "bingoNumberOfTheDay", 86400)
		if err != nil {
			log.Error().
				Str("hostname", r.Host).
				Str("method", r.Method).
				Str("proto", r.Proto).
				Str("remote_ip", r.RemoteAddr).
				Str("path", r.RequestURI).
				Str("user-agent", r.UserAgent()).
				Int("status", http.StatusInternalServerError).
				Msg(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "Oops something wrong happened...")
			return
		}
		log.Info().
			Str("hostname", r.Host).
			Str("method", r.Method).
			Str("proto", r.Proto).
			Str("remote_ip", r.RemoteAddr).
			Str("path", r.RequestURI).
			Str("user-agent", r.UserAgent()).
			Int("status", http.StatusOK).
			Msg("New bingoNumberOfTheDay succesfully generated: " + strconv.Itoa(bingoNumberOfTheDay))
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, bingoNumberOfTheDay)
	} else {
		log.Info().
			Str("hostname", r.Host).
			Str("method", r.Method).
			Str("proto", r.Proto).
			Str("remote_ip", r.RemoteAddr).
			Str("path", r.RequestURI).
			Str("user-agent", r.UserAgent()).
			Int("status", http.StatusInternalServerError).
			Msg("bingoNumberOfTheDay already generated")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "bingoNumberOfTheDay already exists")
	}
}
