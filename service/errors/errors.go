package errors

type NilConfigError struct{}

func (e *NilConfigError) Error() string {
	return "MainConfig can not be nil"
}
