package server

import (
	"strconv"

	"github.com/b2network/b2-indexer/internal/config"
	logger "github.com/b2network/b2-indexer/pkg/log"
)

// ServerContextKey defines the context key used to retrieve a server.Context from
// a command's Context.
const ServerContextKey = "server.context"

// server context
type Context struct {
	cfg    *config.Config
	Logger logger.Logger
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
		logger.New(logger.NewOptions()),
	)
}

func NewContext(cfg *config.Config, logger logger.Logger) *Context {
	return &Context{cfg, logger}
}
