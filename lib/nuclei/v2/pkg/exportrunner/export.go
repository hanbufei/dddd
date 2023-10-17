package exportrunner

import (
	"github.com/projectdiscovery/nuclei/v2/internal/runner"
	"github.com/projectdiscovery/nuclei/v2/pkg/types"
)

func ExportRunnerConfigureOptions() error {
	return runner.ConfigureOptions()
}

func ExportRunnerParseOptions(options *types.Options) {
	runner.ParseOptions(options)
}

func ExportRunnerNew(options *types.Options) (*runner.Runner, error) {
	return runner.New(options)
}
