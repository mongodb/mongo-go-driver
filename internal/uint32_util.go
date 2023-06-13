package internal

// CompareUint32Ptr is a piecewise function with the following return
// conditions:
//
// (1)  2, ptr1 != nil AND ptr2 == nil
// (2)  1, ptr1 > ptr2
// (3)  0, ptr1 == ptr2
// (4) -1, ptr1 < ptr2
// (5) -2, ptr1 == nil AND ptr2 != nil
func CompareUint32Ptr(ptr1, ptr2 *uint32) int {
	if ptr1 == ptr2 {
		// This will catch the double nil or same-pointer cases.
		return 0
	}

	if ptr1 == nil && ptr2 != nil {
		return -2
	}

	if ptr1 != nil && ptr2 == nil {
		return 2
	}

	if *ptr1 > *ptr2 {
		return 1
	}

	if *ptr1 < *ptr2 {
		return -1
	}

	return 0
}
