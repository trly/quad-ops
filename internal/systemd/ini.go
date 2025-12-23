package systemd

import "io"

// WriteUnit serializes a unit file to the given writer using go-ini.
func (u *Unit) WriteUnit(w io.Writer) error {
	_, err := u.File.WriteTo(w)
	return err
}
