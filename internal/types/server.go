package types

type serverContext string

// ServerContextKey defines the context key used to retrieve a server.Context from
// a command's Context.
const (
	ServerContextKey = serverContext("server.context")
	DBContextKey     = serverContext("db.context")
)
