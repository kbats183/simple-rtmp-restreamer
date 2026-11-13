package registry

var (
	ErrStreamNotExist = "StreamNotExist"
)

type StreamNotFoundError struct{}

func (e StreamNotFoundError) Error() string {
	return ErrStreamNotExist
}
