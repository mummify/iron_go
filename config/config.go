// config helper for cache, mq, and worker
package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

  mummify "github.com/mummify/ptah/config"
)

// Contains the configuration for an iron.io service.
// An empty instance is not usable
type Settings struct {
	Token      string `json:"token,omitempty"`
	ProjectId  string `json:"project_id,omitempty"`
	Host       string `json:"host,omitempty"`
	Scheme     string `json:"scheme,omitempty"`
	Port       uint16 `json:"port,omitempty"`
	ApiVersion string `json:"api_version,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

var (
	debug     = false
	goVersion = runtime.Version()
	Presets   = map[string]Settings{
		"worker": Settings{
			Scheme:     "https",
			Port:       443,
			ApiVersion: "2",
			Host:       "worker-aws-us-east-1.iron.io",
			UserAgent:  "iron_go/worker 2.0 (Go " + goVersion + ")",
		},
		"mq": Settings{
			Scheme:     "https",
			Port:       443,
			ApiVersion: "1",
			Host:       "mq-aws-us-east-1.iron.io",
			UserAgent:  "iron_go/mq 1.0 (Go " + goVersion + ")",
		},
		"cache": Settings{
			Scheme:     "https",
			Port:       443,
			ApiVersion: "1",
			Host:       "cache-aws-us-east-1.iron.io",
			UserAgent:  "iron_go/cache 1.0 (Go " + goVersion + ")",
		},
	}
)

func dbg(v ...interface{}) {
	if debug {
		fmt.Fprintln(os.Stderr, v...)
	}
}

// Config gathers configuration from env variables and json config files.
// Examples of fullProduct are "iron_worker", "iron_cache", "iron_mq".
func Config(fullProduct string) (settings Settings) {
	if os.Getenv("IRON_CONFIG_DEBUG") != "" {
		debug = true
		dbg("debugging of config enabled")
	}
	pair := strings.SplitN(fullProduct, "_", 2)
	if len(pair) != 2 {
		panic("Invalid product name, has to use prefix.")
	}
	family, product := pair[0], pair[1]

	base, found := Presets[product]

	if !found {
		base = Settings{
			Scheme:     "https",
			Port:       443,
			ApiVersion: "1",
			Host:       product + "-aws-us-east-1.iron.io",
			UserAgent:  "iron_go",
		}
	}

	base.globalConfig(family, product)
	base.globalEnv(family, product)
	base.productEnv(family, product)
	base.localConfig(family, product)

	return base
}

func (s *Settings) globalConfig(family, product string) {
	if u, err := user.Current(); err == nil {
		path := filepath.Join(u.HomeDir, ".iron.json")
		s.UseConfigFile(family, product, path)
	}
}

// The environment variables the scheme looks for are all of the same formula:
// the camel-cased product name is switched to an underscore (“IronWorker”
// becomes “iron_worker”) and converted to be all capital letters. For the
// global environment variables, “IRON” is used by itself. The value being
// loaded is then joined by an underscore to the name, and again capitalised.
// For example, to retrieve the OAuth token, the client looks for “IRON_TOKEN”.
func (s *Settings) globalEnv(family, product string) {
	eFamily := strings.ToUpper(family) + "_"
	s.commonEnv(eFamily)
}

// In the case of product-specific variables (which override global variables),
// it would be “IRON_WORKER_TOKEN” (for IronWorker).
func (s *Settings) productEnv(family, product string) {
	eProduct := strings.ToUpper(family) + "_" + strings.ToUpper(product) + "_"
	s.commonEnv(eProduct)
}

func (s *Settings) localConfig(family, product string) {
	s.UseConfigFile(family, product, "iron.json")
  s.Token = mummify.MustGetString("iron.io.token")
  s.ProjectId = mummify.MustGetString("iron.io.project.id")
}

func (s *Settings) commonEnv(prefix string) {
	if token := os.Getenv(prefix + "TOKEN"); token != "" {
		s.Token = token
		dbg("env has TOKEN:", s.Token)
	}
	if pid := os.Getenv(prefix + "PROJECT_ID"); pid != "" {
		s.ProjectId = pid
		dbg("env has PROJECT_ID:", s.ProjectId)
	}
	if host := os.Getenv(prefix + "HOST"); host != "" {
		s.Host = host
		dbg("env has HOST:", s.Host)
	}
	if scheme := os.Getenv(prefix + "SCHEME"); scheme != "" {
		s.Scheme = scheme
		dbg("env has SCHEME:", s.Scheme)
	}
	if port := os.Getenv(prefix + "PORT"); port != "" {
		n, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			panic(err)
		}
		s.Port = uint16(n)
		dbg("env has PORT:", s.Port)
	}
	if vers := os.Getenv(prefix + "API_VERSION"); vers != "" {
		s.ApiVersion = vers
		dbg("env has API_VERSION:", s.ApiVersion)
	}
}

// Load and merge the given JSON config file.
func (s *Settings) UseConfigFile(family, product, path string) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		dbg("tried to", err, ": skipping")
		return
	}

	data := map[string]interface{}{}
	err = json.Unmarshal(content, &data)
	if err != nil {
		panic("Invalid JSON in " + path + ": " + err.Error())
	}

	dbg("config in", path, "found")
	s.UseConfigMap(data)

	ipData, found := data[family+"_"+product]
	if found {
		pData := ipData.(map[string]interface{})
		s.UseConfigMap(pData)
	}
}

// Merge the given data into the settings.
func (s *Settings) UseConfigMap(data map[string]interface{}) {
	if token, found := data["token"]; found {
		s.Token = token.(string)
		dbg("config has token:", s.Token)
	}
	if projectId, found := data["project_id"]; found {
		s.ProjectId = projectId.(string)
		dbg("config has project_id:", s.ProjectId)
	}
	if host, found := data["host"]; found {
		s.Host = host.(string)
		dbg("config has host:", s.Host)
	}
	if prot, found := data["scheme"]; found {
		s.Scheme = prot.(string)
		dbg("config has scheme:", s.Scheme)
	}
	if port, found := data["port"]; found {
		s.Port = uint16(port.(float64))
		dbg("config has port:", s.Port)
	}
	if vers, found := data["api_version"]; found {
		s.ApiVersion = vers.(string)
		dbg("config has api_version:", s.ApiVersion)
	}
	if agent, found := data["user_agent"]; found {
		s.UserAgent = agent.(string)
		dbg("config has user_agent:", s.UserAgent)
	}
}
