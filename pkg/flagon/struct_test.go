package flagon

import (
	"flag"
	"fmt"
)

func ExampleStruct() {
	// CAPSExample is going to be embedded in Conf
	type CAPSExample struct {
		InHere int `usage:"struct fields' fields get prefixed with the containing field's name"`
	}

	// AnotherEmbed is also going to be embedded
	type AnotherEmbed struct {
		Ding bool `usage:"does stuff"`
		Another int `usage:"frob"`
	}

	type Conf struct {
		// this field has the flag -boolFlag
		BoolFlag  bool    `usage:"this flag is a boolean"`
		FloatFlag float64 `usage:"and this one is a float"`
		// flags for CAPSExample are prefixed with "CAPSExample", eg. -CAPSExample.inHere
		CAPSExample

		// flags for AnotherEmbed are prefixed with "anotherEmbed", eg. -anotherEmbed.ding
		AnotherEmbed
	}

	conf := &Conf{
		BoolFlag:    true,
		FloatFlag:   1.0,
		CAPSExample: CAPSExample{1},
		// fields in AnotherEmbed will default to their zero values, false and 0
	}
	// create all the flags and "bind" them to conf's fields. The values in conf at the time when
	// Struct is called will be used as defaults
	Struct(conf)
	flag.Parse()
	// conf now has values passed in from the command line

	flagSet := flag.CommandLine

	// get the usage text for the flag -CAPSExample.inHere
	f := flagSet.Lookup("CAPSExample.inHere")
	fmt.Print(f.Usage)

	// Output: struct fields' fields get prefixed with the containing field's name
}
