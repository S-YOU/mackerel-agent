package spec

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/mackerelio/mackerel-agent/logging"
)

// This Generator collects metadata about cloud instances.
// Currently only EC2 is supported.
// EC2: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AESDG-chapter-instancedata.html
// GCE: https://developers.google.com/compute/docs/metadata
// DigitalOcean: https://developers.digitalocean.com/metadata/

// CloudGenerator definition
type CloudGenerator struct {
	baseURL *url.URL
}

// Key is a root key for the generator.
func (g *CloudGenerator) Key() string {
	return "cloud"
}

var cloudLogger = logging.GetLogger("spec.cloud")

var ec2BaseURL, gcpBaseURL, digitalOceanBaseURL *url.URL

func init() {
	ec2BaseURL, _ = url.Parse("http://169.254.169.254/latest/meta-data")
	gcpBaseURL, _ = url.Parse("http://metadata.google.internal/computeMetadata/v1")
	digitalOceanBaseURL, _ = url.Parse("http://169.254.169.254/metadata/v1") // has not been yet used
}

// NewCloudGenerator creates a Cloud Generator instance with specified baseurl.
func NewCloudGenerator(baseurl string) (*CloudGenerator, error) {
	if baseurl == "" {
		baseurl = "http://169.254.169.254/latest/meta-data"
	}
	u, err := url.Parse(baseurl)
	if err != nil {
		return nil, err
	}
	return &CloudGenerator{u}, nil
}

// Generate collects metadata from cloud platform.
func (g *CloudGenerator) Generate() (interface{}, error) {

	timeout := time.Duration(100 * time.Millisecond)
	client := http.Client{
		Timeout: timeout,
	}

	metadataKeys := []string{
		"instance-id",
		"instance-type",
		"placement/availability-zone",
		"security-groups",
		"ami-id",
		"hostname",
		"local-hostname",
		"public-hostname",
		"local-ipv4",
		"public-keys",
		"public-ipv4",
		"reservation-id",
	}

	metadata := make(map[string]string)

	for _, key := range metadataKeys {
		resp, err := client.Get(g.baseURL.String() + "/" + key)
		if err != nil {
			cloudLogger.Debugf("This host may not be running on EC2. Error while reading '%s'", key)
			return nil, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				cloudLogger.Errorf("Results of requesting metadata cannot be read: '%s'", err)
				break
			}
			metadata[key] = string(body)
			cloudLogger.Debugf("results %s:%s", key, string(body))
		} else {
			cloudLogger.Warningf("Status code of the result of requesting metadata '%s' is '%d'", key, resp.StatusCode)
		}
	}

	results := make(map[string]interface{})
	results["provider"] = "ec2"
	results["metadata"] = metadata

	return results, nil
}
