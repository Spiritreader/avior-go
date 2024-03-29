package comparator

import (
	"sort"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/media"
)

const (
	DISC string = "no encode"
	NOCH string = "noch nicht"
	REPL string = "allow replacement"
)

type Module interface {
	// Run executes a module
	//
	// file[0] must be the new file, file[1] must be the old file
	// Returns name of the module, the result and the reason
	Run(...media.File) (string, string, string)
	Priority() int
	Name() string
	Init(config.ModuleConfig)
}

// Initializes all modules for duplicate checking
func InitDupeModules() []Module {
	modules := []Module{
		&DuplicateLengthCheckModule{},
		&ErrorReplaceModule{},
		&LegacyModule{},
		&AgeModule{},
		&AudioModule{},
		&ResolutionModule{},
		&LogMatchModule{},
		&SizeApproxModule{},
	}
	return initModules(modules)
}

// Initialize all modules for single file checking
func InitStandaloneModules() []Module {
	modules := []Module{
		&LengthModule{},
		&MaxSizeModule{},
		&ErrorSkipModule{},
	}
	return initModules(modules)
}

func initModules(modules []Module) []Module {
	cfg := config.Instance()
	for idx := range modules {
		modules[idx].Init(cfg.Local.Modules[modules[idx].Name()])
	}
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Priority() > modules[j].Priority()
	})
	return modules
}
