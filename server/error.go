package server

type AuthError struct {
	Origin error
}

func NewAuthError(err error) *AuthError {
	return &AuthError{Origin: err}
}

func (err AuthError) Error() string {
	return err.Origin.Error()
}
