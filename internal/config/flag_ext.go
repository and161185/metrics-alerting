// internal/config/flagx.go
package config

import "strconv"

type strFlag struct {
	v   string
	set bool
}

func (f *strFlag) String() string     { return f.v }
func (f *strFlag) Set(s string) error { f.v, f.set = s, true; return nil }

type intFlag struct {
	v   int
	set bool
}

func (f *intFlag) String() string { return "" }
func (f *intFlag) Set(s string) error {
	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	f.v, f.set = i, true
	return nil
}

type boolFlag struct {
	v   bool
	set bool
}

func (f *boolFlag) String() string { return "" }
func (f *boolFlag) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	f.v, f.set = b, true
	return nil
}
