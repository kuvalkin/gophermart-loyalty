package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	stdLog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/order/accrual"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
	balanceStorage "github.com/kuvalkin/gophermart-loyalty/internal/storage/balance"
	"github.com/kuvalkin/gophermart-loyalty/internal/storage/balance/withdrawals"
	orderStorage "github.com/kuvalkin/gophermart-loyalty/internal/storage/order"
	userStorage "github.com/kuvalkin/gophermart-loyalty/internal/storage/user"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/config"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/database"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/event"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/transaction"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport"
)

func main() {
	defer event.Release()

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

	conf, err := config.Resolve()
	if err != nil {
		log.Logger().Fatalw("failed to resolve config", "error", err)
		os.Exit(1)
	}

	db, err := initDB(conf)
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

	userService, err := initUserService(conf, db)
	if err != nil {
		log.Logger().Fatalw("failed to initialize user service", "error", err)
		os.Exit(1)
	}

	orderService, poller, err := initOrderService(conf, db)
	if err != nil {
		log.Logger().Fatalw("failed to initialize order service", "error", err)
		os.Exit(1)
	}
	defer func() {
		err := poller.Close()
		if err != nil {
			log.Logger().Fatalw("failed to close poller", "error", err)
		}
	}()

	balanceService, err := initBalanceService(conf, db)
	if err != nil {
		log.Logger().Fatalw("failed to initialize balance service", "error", err)
		os.Exit(1)
	}
	defer func() {
		err := balanceService.Close()
		if err != nil {
			log.Logger().Fatalw("failed to close balance service", "error", err)
		}
	}()

	serv := transport.NewServer(conf, &transport.Services{
		User:    userService,
		Order:   orderService,
		Balance: balanceService,
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

func initUserService(conf *config.Config, db *sql.DB) (user.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), conf.DatabaseTimeout)
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

	return user.NewService(
		userStorage.NewDatabaseRepository(db, conf.DatabaseTimeout),
		&user.Options{
			TokenSecret:           tokenSecret,
			PasswordSalt:          passwordHash,
			MinPasswordLength:     conf.MinPasswordLength,
			TokenExpirationPeriod: conf.TokenExpirationPeriod,
		},
	)
}

func initOrderService(conf *config.Config, db *sql.DB) (order.Service, io.Closer, error) {
	poller, err := accrual.NewPoller(conf.AccrualSystemAddress, conf.AccrualTimeout, conf.AccrualMaxRetries, conf.AccrualMaxRetryPeriod)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize poller for order service: %w", err)
	}

	service := order.NewService(
		orderStorage.NewDatabaseRepository(db, conf.DatabaseTimeout),
		poller,
	)

	ctx, cancel := context.WithTimeout(context.Background(), conf.DatabaseTimeout)
	defer cancel()
	unprocessed, err := orderStorage.GetUnprocessedOrders(ctx, db)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve unprocessed orders: %w", err)
	}

	for _, uo := range unprocessed {
		err = service.AddToProcessQueue(uo.Number, uo.UserID, uo.CurrentStatus)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to add unprocessed order to queue: %w", err)
		}
	}

	return service, poller, nil
}

func initBalanceService(conf *config.Config, db *sql.DB) (balance.Service, error) {
	service, err := balance.NewService(
		balanceStorage.NewDatabaseRepository(db, conf.DatabaseTimeout),
		withdrawals.NewDatabaseRepository(db, conf.DatabaseTimeout),
		transaction.NewDatabaseTransactionProvider(db),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize balance service: %w", err)
	}

	return service, nil
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
