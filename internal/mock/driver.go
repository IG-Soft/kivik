package mock

import (
	"github.com/IG-Soft/kivik/v3/driver"
)

// Driver mocks a Kivik Driver.
type Driver struct {
	NewClientFunc func(name string) (driver.Client, error)
}

var _ driver.Driver = &Driver{}

// NewClient calls d.NewClientFunc
func (d *Driver) NewClient(name string) (driver.Client, error) {
	return d.NewClientFunc(name)
}
