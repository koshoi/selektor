package flag

type FlagValue struct {
	value    string
	FlagType string
}

func (fv *FlagValue) Set(str string) error {
	fv.value = str
	return nil
}

func (fv *FlagValue) Type() string {
	return fv.FlagType
}

func (fv *FlagValue) String() string {
	return fv.value
}
