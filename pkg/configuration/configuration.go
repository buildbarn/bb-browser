package configuration

import (
	"os"

	pb "github.com/buildbarn/bb-browser/pkg/proto/configuration/bb_browser"
	"github.com/golang/protobuf/jsonpb"
)

// GetBrowserConfiguration reads the configuration from file and fill in default values.
func GetBrowserConfiguration(path string) (*pb.BrowserConfiguration, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var browserConfiguration pb.BrowserConfiguration
	if err := jsonpb.Unmarshal(file, &browserConfiguration); err != nil {
		return nil, err
	}
	setDefaultBrowserValues(&browserConfiguration)
	return &browserConfiguration, err
}

func setDefaultBrowserValues(browserConfiguration *pb.BrowserConfiguration) {
	if browserConfiguration.MaximumMessageSizeBytes == 0 {
		browserConfiguration.MaximumMessageSizeBytes = 16*1024*1024
	}
	if browserConfiguration.ListenAddress == "" {
		browserConfiguration.ListenAddress = ":80"
	}
}

