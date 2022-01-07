package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
	redigotrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gomodule/redigo"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	tracer.Start(
		tracer.WithEnv("prod"),
		tracer.WithService("bingo-generator"),
		tracer.WithServiceVersion("v1.0"),
	)
	defer tracer.Stop()
	r := muxtrace.NewRouter(muxtrace.WithServiceName("bingo-generator"))
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
			return redigotrace.Dial("tcp", "redis-master:6379",
				redigotrace.WithServiceName("redis"),
			)
		},
	}

	conn := pool.Get()
	defer conn.Close()

	root, ctx := tracer.StartSpanFromContext(context.Background(), "parent.request",
		tracer.ServiceName("bingo-generator"),
		tracer.ResourceName("redis"),
	)
	defer root.Finish()
	_, err := conn.Do("AUTH", os.Getenv("REDIS_PASSWORD"), ctx)
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

	exists, err := redis.Int(conn.Do("EXISTS", "bingoNumberOfTheDay", ctx))
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
		_, err = conn.Do("SET", "bingoNumberOfTheDay", bingoNumberOfTheDay, ctx)
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
		_, err = conn.Do("EXPIRE", "bingoNumberOfTheDay", 86400, ctx)
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
