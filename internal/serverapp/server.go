package serverapp

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/fr3shw3b/ably-protocol-exercise/pkg/config"
	"github.com/fr3shw3b/ably-protocol-exercise/pkg/server"
	"github.com/fr3shw3b/ably-protocol-exercise/pkg/sessions"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/joho/godotenv"
)

func Run(port int) error {
	err := godotenv.Load(".env.server")
	if err != nil {
		log.Fatal("Failed to load environment variables: ", err)
	}

	router := mux.NewRouter()
	httpSrv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadTimeout:       1 * time.Second,
		WriteTimeout:      1 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		Handler:           router,
	}

	conf, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration for server: ", err)
	}

	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02T15:04:05.999999999Z07:00"
	customFormatter.FullTimestamp = true
	logger := logrus.New()
	logger.SetFormatter(customFormatter)
	logLevel, err := logrus.ParseLevel(conf.LogLevel)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	store := sessions.NewInMemoryStore(
		&sessions.InMemoryStoreParams{
			ExpireAfterIdleTime: conf.SessionStateIdleTimeExpiry,
		},
		logger,
	)

	srv := server.NewDefaultServer(
		&server.ServerParams{
			SequenceMessageInterval: conf.SequenceMessageInterval,
		},
		store,
		logger,
	)
	router.Handle("/", srv)

	log.Printf("Server listening on port %d ... \n", port)
	return httpSrv.ListenAndServe()
}
