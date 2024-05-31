package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/codepnw/api-clean-arch/internal/config"
	"github.com/codepnw/api-clean-arch/internal/user"
	"github.com/codepnw/api-clean-arch/internal/user/db"
	"github.com/codepnw/api-clean-arch/pkg/client/mongodb"
	"github.com/codepnw/api-clean-arch/pkg/logging"
	"github.com/julienschmidt/httprouter"
)

func main() {
	logger := logging.GetLogger()
	logger.Info("create router")
	router := httprouter.New()

	cfg := config.GetConfig()
	cfgMongo := cfg.MongoDB

	mongoDBClient, err := mongodb.NewClient(context.Background(), cfgMongo.Host, cfgMongo.Port, cfgMongo.Username, cfgMongo.Password, cfgMongo.Database, cfgMongo.AuthDB)
	if err != nil {
		panic(err)
	}

	storage := db.NewStorage(mongoDBClient, cfg.MongoDB.Collection, logger)

	// Start Debug User
	users, err := storage.FindAll(context.Background())
	fmt.Println(users)

	user1 := user.User{
		ID:           "",
		Email:        "user1@mail.com",
		Username:     "myuser1",
		PasswordHash: "123123d",
	}

	user1ID, err := storage.Create(context.Background(), user1)
	if err != nil {
		panic(err)
	}
	logger.Info(user1ID)

	user2 := user.User{
		ID:           "",
		Email:        "user2@mail.com",
		Username:     "user2",
		PasswordHash: "123123",
	}
	user2ID, err := storage.Create(context.Background(), user2)
	if err != nil {
		panic(err)
	}
	logger.Info(user2ID)

	user2Found, err := storage.FindOne(context.Background(), user2ID)
	if err != nil {
		panic(err)
	}
	fmt.Println(user2Found)

	user2Found.Email = "newmailuser2@mail.com"
	err = storage.Update(context.Background(), user2Found)
	if err != nil {
		panic(err)
	}

	err = storage.Delete(context.Background(), user2ID)
	if err != nil {
		panic(err)
	}

	_, err = storage.FindOne(context.Background(), user2ID)
	if err == nil {
		panic(err)
	}
	// End Debug User

	handler := user.NewHandler(logger)
	handler.Register(router)

	start(router, cfg)
}

func start(router *httprouter.Router, cfg *config.Config) {
	logger := logging.GetLogger()
	logger.Info("start application")

	var listener net.Listener
	var listenErr error

	if cfg.Listen.Type == "sock" {
		logger.Info("detect app path")

		appDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			logger.Fatal(err)
		}
		logger.Info("create socket")
		socketPath := path.Join(appDir, "app.sock")

		logger.Info("listen unix socket")
		listener, listenErr = net.Listen("unix", socketPath)
		logger.Infof("server is listening unix socket: %s", socketPath)
	} else {
		logger.Info("listen tcp")
		listener, listenErr = net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.Listen.BindIP, cfg.Listen.Port))
		logger.Infof("server is listening port %s:%s", cfg.Listen.BindIP, cfg.Listen.Port)
	}

	if listenErr != nil {
		logger.Fatal(listenErr)
	}

	server := &http.Server{
		Handler:      router,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	logger.Fatal(server.Serve(listener))
}
