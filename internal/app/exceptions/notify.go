package exceptions

const (
	Success        = 200
	SystemError    = 1
	ParameterError = 4

	RequestTypeNonsupport   = 2001
	RequestDetailUnmarshal  = 2002
	RequestDetailParameter  = 2003
	RequestDetailToMismatch = 2004
	IPWhiteList             = 2005
	RequestDetailAmount     = 2006
)
