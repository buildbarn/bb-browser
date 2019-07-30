package configuration

import (
	pb "github.com/buildbarn/bb-browser/pkg/proto/configuration/bb_browser"
	"github.com/buildbarn/bb-storage/pkg/util"
)

// GetBrowserConfiguration reads the configuration from file and fill in default values.
func GetBrowserConfiguration(path string) (*pb.BrowserConfiguration, error) {
	var browserConfiguration pb.BrowserConfiguration
	if err := util.UnmarshalConfigurationFromFile(path, &browserConfiguration); err != nil {
		return nil, util.StatusWrap(err, "Failed to retrieve configuration")
	}
	setDefaultBrowserValues(&browserConfiguration)
	return &browserConfiguration, nil
}

func setDefaultBrowserValues(browserConfiguration *pb.BrowserConfiguration) {
	if browserConfiguration.MaximumMessageSizeBytes == 0 {
		browserConfiguration.MaximumMessageSizeBytes = 16 * 1024 * 1024
	}
	if browserConfiguration.ListenAddress == "" {
		browserConfiguration.ListenAddress = ":80"
	}
}
