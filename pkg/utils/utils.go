package utils

import (
	"fmt"
	"math/rand"
)

func GenId() string {
	return fmt.Sprintf("%d", rand.Uint64())
}
