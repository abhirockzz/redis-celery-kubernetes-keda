package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	rdata "github.com/Pallinder/go-randomdata"
	"github.com/gocelery/gocelery"
	"github.com/gomodule/redigo/redis"
)

const (
	redisHostEnvVar     = "REDIS_HOST"
	redisPasswordEnvVar = "REDIS_PASSWORD"
	taskNameEnvVar      = "TASK_NAME"
)

var (
	redisHost     string
	redisPassword string
	taskName      string
)

func init() {
	redisHost = os.Getenv(redisHostEnvVar)
	redisPassword = os.Getenv(redisPasswordEnvVar)
	if redisHost == "" || redisPassword == "" {
		log.Fatal("redis host or password missing")
	}

	taskName := os.Getenv(taskNameEnvVar)
	if taskName == "" {
		taskName = "users.registration"
	}
}

func main() {
	redisPool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisHost, redis.DialPassword(redisPassword), redis.DialUseTLS(true))

			if err != nil {
				return nil, err
			}
			return c, err
		},
	}

	celeryClient, err := gocelery.NewCeleryClient(
		gocelery.NewRedisBroker(redisPool),
		&gocelery.RedisCeleryBackend{Pool: redisPool},
		1,
	)

	if err != nil {
		log.Fatal(err)
	}

	exit := make(chan os.Signal)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	closed := false

	go func() {
		log.Println("celery producer started...")

		for !closed {
			_, err := celeryClient.Delay(taskName, rdata.FullName(rdata.RandomGender), rdata.Email(), rdata.Address())
			if err != nil {
				panic(err)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	<-exit
	log.Println("exit signalled")
	closed = true
	log.Println("celery producer stopped. bye")
}
