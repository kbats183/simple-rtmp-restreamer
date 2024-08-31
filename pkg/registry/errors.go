package registry

import "fmt"

var (
	StreamNotExist = "StreamNotExist"
)

type StreamNotFound struct{}

func (e StreamNotFound) Error() string {
	return fmt.Sprintf("%s", StreamNotExist)
}
