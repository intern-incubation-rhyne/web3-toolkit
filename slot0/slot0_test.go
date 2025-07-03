package slot0

import (
	"fmt"
	"testing"
)

const ca = "0x5764a6f2212d502bc5970f9f129ffcd61e5d7563"

func TestQuery(t *testing.T) {
	slotZero := Query(ca)
	fmt.Println(slotZero)
	price := PriceFromSqrtPriceX96(slotZero.SqrtPriceX96)
	fmt.Printf("\nPrice: %v\n", price)
}
