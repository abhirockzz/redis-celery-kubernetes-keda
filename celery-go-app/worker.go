package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gocelery/gocelery"
	"github.com/gomodule/redigo/redis"
	uuid "github.com/hashicorp/go-uuid"
)

const (
	redisHostEnvVar     = "REDIS_HOST"
	redisPasswordEnvVar = "REDIS_PASSWORD"
	hashNamePrefix      = "users:"
	taskNameEnvVar      = "TASK_NAME"
)

var (
	redisHost     string
	redisPassword string
	workerID      string
	taskName      string
)

func init() {
	redisHost = os.Getenv(redisHostEnvVar)
	redisPassword = os.Getenv(redisPasswordEnvVar)
	if redisHost == "" || redisPassword == "" {
		log.Fatal("redis host or password missing")
	}

	workerID, _ = uuid.GenerateUUID()

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

	save := func(name, email, address string) string {
		//log.Println("got info - ", name, email, address)
		sleepFor := rand.Intn(9) + 1
		log.Printf("worker %s sleeping for %v sec", workerID, sleepFor)
		time.Sleep(time.Duration(sleepFor) * time.Second)

		info := map[string]string{"name": name, "email": email, "address": address, "worker": workerID, "processed_at": time.Now().UTC().String()}
		hashName := hashNamePrefix + strconv.Itoa(rand.Intn(1000)+1)

		_, err := redisPool.Get().Do("HSET", redis.Args{}.Add(hashName).AddFlat(info)...)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("saved user info in HASH", hashName)
		return hashName
	}

	celeryClient.Register(taskName, save)

	go func() {
		celeryClient.StartWorker()
		log.Println("celery worker started")
	}()

	exit := make(chan os.Signal)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	<-exit
	log.Println("exit signalled")

	celeryClient.StopWorker()
	log.Println("celery worker stopped. bye")
}
