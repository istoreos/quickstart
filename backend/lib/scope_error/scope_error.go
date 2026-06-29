package scope_error

type ScopeErr struct {
	error
	code  int
	scope string
}

func (scopeErr *ScopeErr) Unwrap() error {
	return scopeErr.error
}

func (scopeErr *ScopeErr) Scope() string {
	return scopeErr.scope
}

func (scopeErr *ScopeErr) Code() int {
	return scopeErr.code
}

func (scopeErr *ScopeErr) Error() string {
	return scopeErr.error.Error()
}

func (scopeErr *ScopeErr) Cause() error {
	return scopeErr.error
}

func NewScopeErr(err error, code int, scope string) *ScopeErr {
	return &ScopeErr{
		error: err,
		code:  code,
		scope: scope,
	}
}
