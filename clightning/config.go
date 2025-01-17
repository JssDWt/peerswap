package clightning

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elementsproject/glightning/glightning"
	"github.com/elementsproject/peerswap/log"
	"github.com/pelletier/go-toml/v2"
)

const (
	defaultRpcHost          = "http://127.0.0.1"
	defaultBitcoinSubDir    = ".bitcoin"
	defaultElementsSubDir   = ".elements"
	defaultCookieFile       = ".cookie"
	defaultLiquidWalletName = "peerswap"
	dbName                  = "swaps"
	defaultPolicyFileName   = "policy.conf"
	defaultConfigFileName   = "peerswap.conf"
	defaultPeerswapSubDir   = "peerswap"
)

type BitcoinConf struct {
	RpcUser         string
	RpcPassword     string
	RpcPasswordFile string
	RpcHost         string
	RpcPort         uint
	Network         string
	DataDir         string
}

type LiquidConf struct {
	RpcUser         string
	RpcPassword     string
	RpcPasswordFile string
	RpcHost         string
	RpcPort         uint
	RpcWallet       string
	Network         string
	DataDir         string
	Disabled        bool
}

type Config struct {
	LightningDir string
	PeerswapDir  string
	DbPath       string
	PolicyPath   string
	Bitcoin      *BitcoinConf
	Liquid       *LiquidConf
}

func (c *Config) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// SetWorkingDir sets the plugin data directory which is the cln
// main data dir, the current working directory of the plugin.
func SetWorkingDir() Processor {
	return func(c *Config) (*Config, error) {
		var err error
		c.LightningDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}

		return c, nil
	}
}

// SetPeerswapPaths sets the Peerswap dir and the db name. If someone wants to
// have them in a different place they need to symlink to the paths.
// Path to peerswap data-dir: `<lightning-dir>/peerswap`.
// Path to peerswap swaps-db: `<lightning-dir>/peerswap/swaps`.
func SetPeerswapPaths(plugin *glightning.Plugin) Processor {
	return func(c *Config) (*Config, error) {
		c.PeerswapDir = filepath.Join(c.LightningDir, defaultPeerswapSubDir)
		c.DbPath = filepath.Join(c.PeerswapDir, dbName)
		return c, nil
	}
}

// CheckForLegacyClnConfig checks if some cln legacy config is set and
// throws an error if so. This is needed to ensure that people switch
// to the new config file instead of using the legacy cln config. Cln
// is not able to pass config on dynamic plugin start, e.g. when
// peerswap is stopped and restarted while cln keeps running.
// We do not consider Peerswap an `important plugin`.
func CheckForLegacyClnConfig(plugin *glightning.Plugin) Processor {
	return func(c *Config) (*Config, error) {
		var reasons []string

		for _, option := range legacyOptions {
			// We don't need to respect the error here has we are only interested in
			// valid set configs that we want to add to our reasons.
			v, _ := plugin.GetOption(option)
			if v != "" {
				reasons = append(reasons, fmt.Sprintf("field is set: %s=%s", option, v))
			}
		}

		if reasons != nil {
			log.Infof(
				"Setting config in core lightning config file is deprecated. Please "+
					"use a standalone 'peerswap.conf' file that resides in the plugin dir "+
					"directory of the plugin (%s): %s",
				c.PeerswapDir,
				strings.Join(reasons, ","),
			)
			return nil, fmt.Errorf("illegal use of core lightning config")
		}

		return c, nil
	}
}

// ReadFromFile reads a config toml file. The file is expected to be
// in the running CLN container.
func ReadFromFile() Processor {
	return func(c *Config) (*Config, error) {
		data, err := ioutil.ReadFile(filepath.Join(c.PeerswapDir, defaultConfigFileName))
		if os.IsNotExist(err) {
			return c, nil
		}
		if err != nil {
			return nil, err
		}

		var fileConf struct {
			Bitcoin *BitcoinConf
			Liquid  *LiquidConf
		}

		err = toml.Unmarshal(data, &fileConf)
		if err != nil {
			return nil, err
		}

		if fileConf.Bitcoin != nil {
			c.Bitcoin.RpcUser = fileConf.Bitcoin.RpcUser
			c.Bitcoin.RpcPassword = fileConf.Bitcoin.RpcPassword
			c.Bitcoin.RpcPasswordFile = fileConf.Bitcoin.RpcPasswordFile
			c.Bitcoin.RpcHost = fileConf.Bitcoin.RpcHost
			c.Bitcoin.RpcPort = fileConf.Bitcoin.RpcPort
		}

		if fileConf.Liquid != nil {
			c.Liquid.RpcUser = fileConf.Liquid.RpcUser
			c.Liquid.RpcPassword = fileConf.Liquid.RpcPassword
			c.Liquid.RpcPasswordFile = fileConf.Liquid.RpcPasswordFile
			c.Liquid.RpcHost = fileConf.Liquid.RpcHost
			c.Liquid.RpcPort = fileConf.Liquid.RpcPort
			c.Liquid.RpcWallet = fileConf.Liquid.RpcWallet
			c.Liquid.Disabled = fileConf.Liquid.Disabled
		}

		return c, nil
	}
}

func PeerSwapFallback() Processor {
	return func(c *Config) (*Config, error) {
		if c.PolicyPath == "" {
			c.PolicyPath = filepath.Join(c.PeerswapDir, defaultPolicyFileName)
		}

		if c.Liquid.RpcWallet == "" {
			c.Liquid.RpcWallet = defaultLiquidWalletName
		}

		return c, nil
	}
}

// BitcoinFallbackFromClnConfig
// if no bitcoin config is set at all, try to fall back to cln bitcoin config.
func BitcoinFallbackFromClnConfig(client *ClightningClient) Processor {
	return func(c *Config) (*Config, error) {
		if c.Bitcoin.RpcUser == "" && c.Bitcoin.RpcPassword == "" &&
			c.Bitcoin.RpcPasswordFile == "" && c.Bitcoin.RpcHost == "" &&
			c.Bitcoin.RpcPort == 0 {
			// No bitcoin config is set, we try to fetch it from CLN.
			conf, err := client.glightning.ListConfigs()
			if err != nil {
				return nil, err
			}

			// Parse interface data into struct.
			data, err := json.Marshal(conf)
			if err != nil {
				return nil, err
			}

			var listConfigResponse struct {
				ImportantPlugins []*struct {
					Path    string
					Name    string
					Options map[string]interface{}
				} `json:"important-plugins"`
			}
			err = json.Unmarshal(data, &listConfigResponse)
			if err != nil {
				return nil, err
			}

			// Extract settings from the `bcli` plugin.
			for _, plugin := range listConfigResponse.ImportantPlugins {
				if plugin.Name == "bcli" {
					// Extract network.
					if v, ok := plugin.Options["network"]; ok {
						if v != nil {
							c.Bitcoin.Network = v.(string)
						}
					}
					if _, ok := plugin.Options["mainnet"]; ok {
						c.Bitcoin.Network = "mainnet"
					}
					if _, ok := plugin.Options["testnet"]; ok {
						c.Bitcoin.Network = "testnet"
					}
					if _, ok := plugin.Options["signet"]; ok {
						c.Bitcoin.Network = "signet"
					}
					if c.Bitcoin.Network == "" {
						// Network could not be extracted, try `getinfo`.
						info, err := client.glightning.GetInfo()
						if err != nil {
							return nil, err
						}
						c.Bitcoin.Network = info.Network
						// Ugly hack to rewrite core-lightnings network name to
						// the internal variant: bitcoin ~ mainnet.
						if c.Bitcoin.Network == "bitcoin" {
							c.Bitcoin.Network = "mainnet"
						}
					}

					// Extract rest of the bitcoind config
					if v, ok := plugin.Options["bitcoin-datadir"]; ok {
						if v != nil {
							c.Bitcoin.DataDir = v.(string)
						}
					}
					if v, ok := plugin.Options["bitcoin-rpcuser"]; ok {
						if v != nil {
							c.Bitcoin.RpcUser = v.(string)
						}
					}
					if v, ok := plugin.Options["bitcoin-rpcpassword"]; ok {
						if v != nil {
							c.Bitcoin.RpcPassword = v.(string)
						}
					}
					if v, ok := plugin.Options["bitcoin-rpcconnect"]; ok {
						if v != nil {
							c.Bitcoin.RpcHost = v.(string)
						}
					}
					if v, ok := plugin.Options["bitcoin-rpcport"]; ok {
						if v != nil {
							port, err := strconv.Atoi(v.(string))
							if err != nil {
								return nil, err
							}
							c.Bitcoin.RpcPort = uint(port)
						}
					}
				}
			}
		}
		return c, nil
	}
}

// BitcoinFallback sets default values for empty config options.
func BitcoinFallback() Processor {
	return func(c *Config) (*Config, error) {
		if c.Bitcoin.DataDir == "" {
			// If no data dir is set, use default location `$HOME/.bitcoin`
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			c.Bitcoin.DataDir = filepath.Join(home, defaultBitcoinSubDir)
		}

		if c.Bitcoin.RpcHost == "" {
			c.Bitcoin.RpcHost = defaultRpcHost
		}

		if c.Bitcoin.RpcPort == 0 {
			c.Bitcoin.RpcPort = defaultBitcoinRpcPort(c.Bitcoin.Network)
		}

		if c.Bitcoin.RpcPassword == "" && c.Bitcoin.RpcUser == "" &&
			c.Bitcoin.RpcPasswordFile == "" {
			// No password, user or cookie set, try to load cookie from default
			// location.
			netdir, err := bitcoinNetDir(c.Bitcoin.Network)
			if err != nil {
				return nil, err
			}
			c.Bitcoin.RpcPasswordFile = filepath.Join(c.Bitcoin.DataDir, netdir, defaultCookieFile)
		}
		return c, nil
	}
}

// ElementsFallback sets default values for empty config options if liquid is
// enabled.
func ElementsFallback() Processor {
	return func(c *Config) (*Config, error) {
		var err error
		if c.Liquid.DataDir == "" {
			// If no data dir is set, use default location `$HOME/.elements`
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			c.Liquid.DataDir = filepath.Join(home, defaultElementsSubDir)
		}

		if c.Liquid.Network == "" {
			c.Liquid.Network, err = liquidNetDir(c.Bitcoin.Network)
			if err != nil {
				return nil, err
			}
		}

		if c.Liquid.RpcHost == "" {
			c.Liquid.RpcHost = defaultRpcHost
		}

		if c.Liquid.RpcPort == 0 {
			c.Liquid.RpcPort = defaultElementsRpcPort(c.Liquid.Network)
		}

		if c.Liquid.RpcPassword == "" && c.Liquid.RpcUser == "" &&
			c.Liquid.RpcPasswordFile == "" {
			// No password, user or cookie set, try to load cookie from default
			// location.
			netdir, err := liquidNetDir(c.Bitcoin.Network)
			if err != nil {
				return nil, err
			}
			c.Liquid.RpcPasswordFile = filepath.Join(c.Liquid.DataDir, netdir, defaultCookieFile)
		}
		return c, nil
	}
}

// BitcoinCookieConnect deflates a cookie file to override rpc user
// and password.
func BitcoinCookieConnect() Processor {
	return func(c *Config) (*Config, error) {
		var err error
		if c.Bitcoin.RpcUser == "" && c.Bitcoin.RpcPassword == "" {
			if c.Bitcoin.RpcPasswordFile == "" {
				return nil, fmt.Errorf("no bitcoin rpc configuration found")
			}
			c.Bitcoin.RpcUser, c.Bitcoin.RpcPassword, err = readCookie(c.Bitcoin.RpcPasswordFile)
			if err != nil {
				log.Infof("Could not read from bitcoin cookie: %s", c.Bitcoin.RpcPasswordFile)
			}
		}
		return c, nil
	}
}

// ElementsCookieConnect deflates a cookie file to override rpc user
// and password.
func ElementsCookieConnect() Processor {
	return func(c *Config) (*Config, error) {
		var err error
		if c.Liquid.RpcUser == "" && c.Liquid.RpcPassword == "" &&
			!c.Liquid.Disabled {
			if c.Liquid.RpcPasswordFile == "" {
				return nil, fmt.Errorf("no liquid rpc configuration found")
			}
			c.Liquid.RpcUser, c.Liquid.RpcPassword, err = readCookie(c.Liquid.RpcPasswordFile)
			if err != nil {
				log.Infof("Could not read from elements cookie: %s", c.Liquid.RpcPasswordFile)
			}
		}
		return c, nil
	}
}

func GetConfig(client *ClightningClient) (*Config, error) {
	pl := &Pipeline{processors: []Processor{}}
	pl = pl.
		Add(SetWorkingDir()).
		Add(SetPeerswapPaths(client.Plugin)).
		Add(CheckForLegacyClnConfig(client.Plugin)).
		Add(ReadFromFile()).
		Add(PeerSwapFallback()).
		Add(BitcoinFallbackFromClnConfig(client)).
		Add(BitcoinFallback()).
		Add(ElementsFallback()).
		Add(BitcoinCookieConnect()).
		Add(ElementsCookieConnect())

	return pl.Run()
}

type Processor func(*Config) (*Config, error)

type Pipeline struct {
	processors []Processor
}

func (p *Pipeline) Add(pr Processor) *Pipeline {
	p.processors = append(p.processors, pr)
	return p
}

func (p *Pipeline) Run() (*Config, error) {
	var err error
	c := &Config{Bitcoin: &BitcoinConf{}, Liquid: &LiquidConf{}}
	for _, pr := range p.processors {
		c, err = pr(c)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func defaultBitcoinRpcPort(network string) uint {
	switch network {
	case "signet":
		return 38332
	case "testnet":
		return 18332
	case "regtest":
		return 18443
	default:
		// mainnet is the default port
		return 8332
	}
}

func defaultElementsRpcPort(network string) uint {
	switch network {
	case "liquidtestnet":
		return 18891
	case "regtest":
		return 18443
	default:
		return 7041
	}
}

func bitcoinNetDir(network string) (string, error) {
	switch network {
	case "mainnet":
		return "", nil
	case "signet":
		return "signet", nil
	case "testnet3":
		return "testnet", nil
	case "regtest":
		return "regtest", nil
	default:
		return "", fmt.Errorf("can not get network dir for bitcoin network: %s", network)
	}
}

func liquidNetDir(network string) (string, error) {
	switch network {
	case "mainnet":
		return "liquidv1", nil
	case "testnet3", "simnet", "signet":
		return "liquidtestnet", nil
	case "regtest":
		return "liquidregtest", nil
	default:
		return "", fmt.Errorf("can not get liquid network dir for bitcoin network: %s", network)
	}
}

func readCookie(path string) (string, string, error) {
	cookieBytes, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	cookie := strings.Split(string(cookieBytes), ":")
	if len(cookie) != 2 {
		return "", "", fmt.Errorf("malformed cookie %v", cookieBytes)
	}

	return cookie[0], cookie[1], nil
}
