package options

import (
	"fmt"
	"os"
)

type Options struct {
	Debug bool
}

func (o *Options) Debugf(f string, values ...interface{}) {
	if !o.Debug {
		return
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] %s\n", fmt.Sprintf(f, values...))
}
