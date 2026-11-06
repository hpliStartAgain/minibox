package box

import "slices"

func (c Config) fault(name string) bool {
	return slices.Contains(c.Faults, name)
}
