package server

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/b2network/b2-indexer/internal/config"
	logger "github.com/b2network/b2-indexer/pkg/log"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"
)

type serverContext string

// ServerContextKey defines the context key used to retrieve a server.Context from
// a command's Context.
const (
	ServerContextKey = serverContext("server.context")
	DBContextKey     = serverContext("db.context")
)

// server context
type Context struct {
	// Viper         *viper.Viper
	Config        *config.Config
	BitcoinConfig *config.BitconConfig
	// Logger        logger.Logger
	// Db *gorm.DB
}

// ErrorCode contains the exit code for server exit.
type ErrorCode struct {
	Code int
}

func (e ErrorCode) Error() string {
	return strconv.Itoa(e.Code)
}

func NewDefaultContext() *Context {
	return NewContext(
		config.DefaultConfig(),
		config.DefaultBitcoinConfig(),
	)
}

func NewContext(cfg *config.Config, btccfg *config.BitconConfig) *Context {
	return &Context{cfg, btccfg}
}

func InterceptConfigsPreRunHandler(cmd *cobra.Command, home string) error {
	cfg, err := config.LoadConfig(home)
	if err != nil {
		return err
	}
	if home != "" {
		cfg.RootDir = home
	}

	bitcoincfg, err := config.LoadBitcoinConfig(home)
	if err != nil {
		return err
	}
	db, err := NewDB(cfg)
	if err != nil {
		return err
	}

	// set db to context
	ctx := context.WithValue(cmd.Context(), DBContextKey, db)
	cmd.SetContext(ctx)

	logger.Init(cfg.LogLevel, cfg.LogFormat)
	serverCtx := NewContext(cfg, bitcoincfg)
	return SetCmdServerContext(cmd, serverCtx)
}

// GetServerContextFromCmd returns a Context from a command or an empty Context
// if it has not been set.
func GetServerContextFromCmd(cmd *cobra.Command) *Context {
	if v := cmd.Context().Value(ServerContextKey); v != nil {
		serverCtxPtr := v.(*Context)
		return serverCtxPtr
	}

	return NewDefaultContext()
}

// SetCmdServerContext sets a command's Context value to the provided argument.
func SetCmdServerContext(cmd *cobra.Command, serverCtx *Context) error {
	v := cmd.Context().Value(ServerContextKey)
	if v == nil {
		return errors.New("server context not set")
	}

	serverCtxPtr := v.(*Context)
	*serverCtxPtr = *serverCtx

	return nil
}

// NewDB creates a new database connection.
// default use postgres driver
func NewDB(cfg *config.Config) (*gorm.DB, error) {
	DB, err := gorm.Open(postgres.Open(cfg.DatabaseSource), &gorm.Config{
		Logger: gormlog.Default.LogMode(gormlog.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}
	// set db conn limit
	sqlDB.SetMaxIdleConns(cfg.DatabaseMaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.DatabaseMaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.DatabaseConnMaxLifetime) * time.Second)
	return DB, nil
}
