package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	stdLog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kuvalkin/gophermart-loyalty/internal/config"
	"github.com/kuvalkin/gophermart-loyalty/internal/database"
	"github.com/kuvalkin/gophermart-loyalty/internal/log"
)

func main() {
	err := log.InitLogger()
	if err != nil {
		stdLog.Fatal(fmt.Errorf("failed to initialize logger: %w", err))
	}

	defer func() {
		err = log.Logger().Sync()
		if err != nil {
			stdLog.Println(fmt.Errorf("failed to sync logger: %w", err))
		}
	}()

	cnf, err := config.Resolve()
	if err != nil {
		log.Logger().Fatalw("failed to resolve config", "error", err)
		os.Exit(1)
	}

	db, err := initDB(cnf)
	if err != nil {
		log.Logger().Fatalw("failed to initialize database", "error", err)
		os.Exit(1)
	}

	defer func() {
		log.Logger().Debug("closing DB connection")

		err := db.Close()
		if err != nil {
			log.Logger().Fatalw("failed to close database", "error", err)
		}
	}()

	serv := &http.Server{
		Addr:    cnf.RunAddress,
		Handler: http.DefaultServeMux, // todo
	}

	go listenAndServe(serv)

	waitForSignalAndShutdown(serv)
}

func initDB(cnf *config.Config) (*sql.DB, error) {
	log.Logger().Debug("connecting to DB")

	ctx, cancel := context.WithTimeout(context.Background(), cnf.DatabaseTimeout)
	defer cancel()

	db, err := database.InitDB(ctx, cnf.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("init db failed: %w", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), cnf.DatabaseTimeout)
	defer cancel()

	err = database.Migrate(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("migrate failed: %w", err)
	}

	return db, nil
}

func listenAndServe(serv *http.Server) {
	log.Logger().Infow("starting server", "address", serv.Addr)

	err := serv.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		log.Logger().Fatalw("error starting server", "error", err)
		os.Exit(1)
	}
}

func waitForSignalAndShutdown(serv *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Waiting (indefinitely) for a signal
	sig := <-stop
	log.Logger().Debugw("received signal", "signal", sig)

	log.Logger().Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := serv.Shutdown(ctx); err != nil {
		log.Logger().Errorw("failed to shutdown server", "error", err)
	}

	log.Logger().Info("server shutdown complete")
}
