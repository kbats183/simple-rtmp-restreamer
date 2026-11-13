package utils

import (
	"fmt"
	"math/rand"
)

func GenID() string {
	return fmt.Sprintf("%d", rand.Uint64())
}
