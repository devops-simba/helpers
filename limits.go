package helpers

const (
	MaxUint = ^uint(0)
	MinUint = uint(0)
	MaxInt  = int(MaxUint >> 1)
	MinInt  = -MaxInt - 1

	// UintBits number of bits in an int(or uint) type.
	// Implementation logic is it will result in (32 << 1) if uint is 64 bit and
	// (32 << 0) if uint is 32 bit
	UintBits = 32 << (MaxUint >> 63)
	IntBits  = UintBits
)
