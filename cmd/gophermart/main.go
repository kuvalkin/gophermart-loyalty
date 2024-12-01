package main

import (
	"context"
	"database/sql"
	"fmt"
	stdLog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kuvalkin/gophermart-loyalty/internal/config"
	"github.com/kuvalkin/gophermart-loyalty/internal/database"
	"github.com/kuvalkin/gophermart-loyalty/internal/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
	userStorage "github.com/kuvalkin/gophermart-loyalty/internal/storage/user"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport"
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

	userService, err := initUserService(db)
	if err != nil {
		log.Logger().Fatalw("failed to initialize user service", "error", err)
		os.Exit(1)
	}

	serv := transport.NewServer(cnf, &transport.Services{
		User: userService,
	})

	go listenAndServe(serv)

	waitForSignalAndShutdown(serv)
	log.Logger().Info("Bye :)")
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

func initUserService(db *sql.DB) (user.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokenSecret, passwordHash, err := userStorage.GetSecretsFromDB(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets from db: %w", err)
	}

	if len(tokenSecret) == 0 {
		tokenSecret, err = user.GenerateTokenSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate token secret: %w", err)
		}

		err = userStorage.WriteTokenSecretToDB(ctx, db, tokenSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to save jwt secret to db: %w", err)
		}
	}

	if passwordHash == "" {
		salt, err := user.GeneratePasswordSalt()
		if err != nil {
			return nil, fmt.Errorf("failed to generate password salt: %w", err)
		}

		err = userStorage.WritePasswordSaltToDB(ctx, db, salt)
		if err != nil {
			return nil, fmt.Errorf("failed to save password salt to db: %w", err)
		}
	}

	return user.NewService(userStorage.NewDatabaseRepository(db), tokenSecret, passwordHash), nil
}

func listenAndServe(serv *transport.Server) {
	err := serv.ListenAndServe()

	if err != nil {
		log.Logger().Fatalw("error starting server", "error", err)
		os.Exit(1)
	}
}

func waitForSignalAndShutdown(serv *transport.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Waiting (indefinitely) for a signal
	sig := <-stop
	log.Logger().Debugw("received signal", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := serv.Shutdown(ctx); err != nil {
		log.Logger().Errorw("failed to shutdown server", "error", err)
	}

	log.Logger().Info("server shutdown complete")
}
