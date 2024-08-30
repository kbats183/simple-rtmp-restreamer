package rtmpserver

import (
	"fmt"
	"math/rand"
)

func genId() string {
	return fmt.Sprintf("%d", rand.Uint64())
}
