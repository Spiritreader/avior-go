package comparator

import (
	"sort"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/structs"
)

const (
	KEEP string = "keep duplicate"
	NOCH string = "noch nicht"
	REPL string = "allow replacement"
)

type Module interface {
	Run() string
	Priority() int
	Name() string
	Init(structs.ModuleConfig)
}

// Initializes all modules
func InitModules() []Module {
	cfg := config.Instance()
	modules := []Module{
		&AgeModule{},
	}
	for idx := range modules {
		modules[idx].Init(cfg.Local.Modules[modules[idx].Name()])
	}
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Priority() > modules[j].Priority()
	})
	return modules
}
