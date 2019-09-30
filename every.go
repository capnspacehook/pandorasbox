package pandorasbox

import (
	"os"
	"sort"

	"github.com/xtgo/set"
)

func ForEveryFlag(fn func(flag int) error) error {
	for _, flag := range EveryFlag() {
		err := fn(flag)
		if err != nil {
			return err
		}
	}
	return nil
}

func ForEveryPermission(fn func(mode os.FileMode) error) error {
	for _, mode := range EveryPermission() {
		err := fn(mode)
		if err != nil {
			return err
		}
	}
	return nil
}

func EveryPermission() []os.FileMode {
	ints := make(sort.IntSlice, 512)
	perms := []uint{OS_READ, OS_WRITE, OS_EX}
	shifts := []uint{OS_OTH_SHIFT, OS_GROUP_SHIFT, OS_USER_SHIFT}

	for i := 0; i < 512; i++ {
		var mode os.FileMode
		for s, shift := range shifts {
			for p, perm := range perms {
				if 1<<uint(p)&(i>>(uint(s)*3)) != 0 {
					mode = os.FileMode(uint(mode) | perm<<shift)
				}
			}
		}
		ints[i] = int(mode)
	}

	sort.Sort(ints)
	n := set.Uniq(ints)
	ints = ints[:n]

	modes := make([]os.FileMode, len(ints))
	for i := range ints {
		modes[i] = os.FileMode(ints[i])
	}

	return modes
}

func EveryFlag() []int {
	flagList := []int{0, os.O_APPEND, os.O_CREATE, os.O_EXCL, os.O_SYNC, os.O_TRUNC}
	accessList := []int{0, os.O_WRONLY, os.O_RDWR, os.O_RDONLY}
	var flags sort.IntSlice

	for _, acc := range accessList {
		flag := acc

		for i := 0; i < 63; i++ {
			if 1<<0&i != 0 {
				flag |= flagList[5]
			}

			if 1<<1&i != 0 {
				flag |= flagList[4]
			}

			if 1<<2&i != 0 {
				flag |= flagList[3]
			}

			if 1<<3&i != 0 {
				flag |= flagList[2]
			}

			if 1<<4&i != 0 {
				flag |= flagList[1]
			}

			if 1<<5&i != 0 {
				flag |= flagList[0]
			}
			flags = append(flags, flag)
		}
	}

	sort.Sort(flags)
	n := set.Uniq(flags)
	flags = flags[:n]
	return flags
}
