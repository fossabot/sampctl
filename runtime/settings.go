package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/Southclaws/sampctl/util"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

var echoMessage = "loading server.cfg generated by sampctl - do not edit this file manually, edit samp.json instead!"

// Config stores the server settings and working directory
type Config struct {
	// Only used internally
	dir *string // local directory that configuration points to

	// Only used to configure sampctl, not used in server.cfg generation
	Version  *string `json:"version,omitempty"`  // SA:MP server binaries version
	Endpoint *string `json:"endpoint,omitempty"` // download endpoint for server binaries

	// Echo - set automatically
	Echo *string `default:"-"             required:"0" json:"echo,omitempty"`

	// Core properties
	Gamemodes     []string `                        json:"gamemodes,omitempty" cfg:"gamemode" numbered:"1"` //
	Filterscripts []string `                        required:"0" json:"filterscripts,omitempty"`            //
	Plugins       []Plugin `                        required:"0" json:"plugins,omitempty"`                  //
	RCONPassword  *string  `required:"1"            json:"rcon_password,omitempty"`                         // changeme
	Port          *int     `default:"8192"          required:"0" json:"port"`                               // 8192
	Hostname      *string  `default:"SA-MP Server"  required:"0" json:"hostname,omitempty"`                 // SA-MP Server
	MaxPlayers    *int     `default:"50"            required:"0" json:"maxplayers"`                         // 50
	Language      *string  `default:"-"             required:"0" json:"language,omitempty"`                 //
	Mapname       *string  `default:"San Andreas"   required:"0" json:"mapname,omitempty"`                  // San Andreas
	Weburl        *string  `default:"www.sa-mp.com" required:"0" json:"weburl,omitempty"`                   // www.sa-mp.com
	GamemodeText  *string  `default:"Unknown"       required:"0" json:"gamemodetext,omitempty"`             // Unknown

	// Network and technical config
	Bind       *string `                        required:"0" json:"bind,omitempty"`       //
	Password   *string `                        required:"0" json:"password,omitempty"`   //
	Announce   *bool   `default:"1"             required:"0" json:"announce,omitempty"`   // 0
	LANMode    *bool   `default:"0"             required:"0" json:"lanmode,omitempty"`    // 0
	Query      *bool   `default:"1"             required:"0" json:"query,omitempty"`      // 0
	RCON       *bool   `default:"0"             required:"0" json:"rcon,omitempty"`       // 0
	LogQueries *bool   `default:"0"             required:"0" json:"logqueries,omitempty"` // 0
	Sleep      *int    `default:"5"             required:"0" json:"sleep,omitempty"`      // 5
	MaxNPC     *int    `default:"0"             required:"0" json:"maxnpc,omitempty"`     // 0

	// Rates and performance
	StreamRate        *int     `default:"1000"          required:"0" json:"stream_rate,omitempty"`       // 1000
	StreamDistance    *float32 `default:"200.0"         required:"0" json:"stream_distance,omitempty"`   // 200.0
	OnFootRate        *int     `default:"30"            required:"0" json:"onfoot_rate,omitempty"`       // 30
	InCarRate         *int     `default:"30"            required:"0" json:"incar_rate,omitempty"`        // 30
	WeaponRate        *int     `default:"30"            required:"0" json:"weapon_rate,omitempty"`       // 30
	ChatLogging       *bool    `default:"1"             required:"0" json:"chatlogging,omitempty"`       // 1
	Timestamp         *bool    `default:"1"             required:"0" json:"timestamp,omitempty"`         // 1
	NoSign            *string  `                        required:"0" json:"nosign,omitempty"`            //
	LogTimeFormat     *string  `default:"[%H:%M:%S]"    required:"0" json:"logtimeformat,omitempty"`     // [%H:%M:%S]
	MessageHoleLimit  *int     `default:"3000"          required:"0" json:"messageholelimit,omitempty"`  // 3000
	MessagesLimit     *int     `default:"500"           required:"0" json:"messageslimit,omitempty"`     // 500
	AcksLimit         *int     `default:"3000"          required:"0" json:"ackslimit,omitempty"`         // 3000
	PlayerTimeout     *int     `default:"10000"         required:"0" json:"playertimeout,omitempty"`     // 10000
	MinConnectionTime *int     `default:"0"             required:"0" json:"minconnectiontime,omitempty"` // 0
	LagCompmode       *int     `default:"1"             required:"0" json:"lagcompmode,omitempty"`       // 1
	ConnseedTime      *int     `default:"300000"        required:"0" json:"connseedtime,omitempty"`      // 300000
	DBLogging         *bool    `default:"0"             required:"0" json:"db_logging,omitempty"`        // 0
	DBLogQueries      *bool    `default:"0"             required:"0" json:"db_log_queries,omitempty"`    // 0
	ConnectCookies    *bool    `default:"1"             required:"0" json:"conncookies,omitempty"`       // 1
	CookieLogging     *bool    `default:"0"             required:"0" json:"cookielogging,omitempty"`     // 1
	Output            *bool    `default:"1"             required:"0" json:"output,omitempty"`            // 1
}

// NewConfigFromEnvironment creates a Config from the given environment which includes a directory
// which is seached for either `samp.json` or `samp.yaml` and environment variable versions of the
// config parameters.
func NewConfigFromEnvironment(dir string) (cfg Config, err error) {
	cfg, err = ConfigFromDirectory(dir)
	if err != nil {
		return
	}

	cfg.dir = &dir

	// Environment variables override samp.json
	cfg.LoadEnvironmentVariables()

	return
}

// ConfigFromDirectory creates a config from a directory by searching for a JSON or YAML file to
// read settings from. If both exist, the JSON file takes precedence.
func ConfigFromDirectory(dir string) (cfg Config, err error) {
	jsonFile := filepath.Join(dir, "samp.json")
	if util.Exists(jsonFile) {
		cfg, err = ConfigFromJSON(jsonFile)
	} else {
		yamlFile := filepath.Join(dir, "samp.yaml")
		if util.Exists(yamlFile) {
			cfg, err = ConfigFromYAML(yamlFile)
		} else {
			err = errors.New("directory does not contain a samp.json or samp.yaml file")
		}
	}

	return
}

// ConfigFromJSON creates a config from a JSON file
func ConfigFromJSON(file string) (cfg Config, err error) {
	var contents []byte
	contents, err = ioutil.ReadFile(file)
	if err != nil {
		err = errors.Wrap(err, "failed to read samp.json")
		return
	}

	err = json.Unmarshal(contents, &cfg)
	if err != nil {
		err = errors.Wrap(err, "failed to unmarshal samp.json")
		return
	}

	return
}

// ConfigFromYAML creates a config from a YAML file
func ConfigFromYAML(file string) (cfg Config, err error) {
	var contents []byte
	contents, err = ioutil.ReadFile(file)
	if err != nil {
		err = errors.Wrap(err, "failed to read samp.json")
		return
	}

	err = yaml.Unmarshal(contents, &cfg)
	if err != nil {
		err = errors.Wrap(err, "failed to unmarshal samp.json")
		return
	}

	return
}

// LoadEnvironmentVariables loads Config fields from environment variables - the variable names are
// simply the `json` tag names uppercased and prefixed with `SAMP_`
func (cfg *Config) LoadEnvironmentVariables() {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		fieldval := v.Field(i)
		stype := t.Field(i)

		if !fieldval.CanSet() {
			continue
		}

		name := "SAMP_" + strings.ToUpper(strings.Split(t.Field(i).Tag.Get("json"), ",")[0])

		value, ok := os.LookupEnv(name)
		if !ok {
			continue
		}

		switch stype.Type.String() {
		case "*string":
			if fieldval.IsNil() {
				v := reflect.ValueOf(value)
				fieldval.Set(reflect.New(v.Type()))
			}
			fieldval.Elem().SetString(value)

		case "[]string":
			// todo: allow filterscripts and plugins via env vars
			fmt.Println("cannot set gamemode via environment variables yet")

		case "*bool":
			valueAsBool, err := strconv.ParseBool(value)
			if err != nil {
				fmt.Printf("warning: environment variable '%s' could not interpret value '%s' as boolean: %v\n", stype.Name, value, err)
			}
			if fieldval.IsNil() {
				v := reflect.ValueOf(valueAsBool)
				fieldval.Set(reflect.New(v.Type()))
			}
			fieldval.Elem().SetBool(valueAsBool)

		case "*int":
			valueAsInt, err := strconv.Atoi(value)
			if err != nil {
				fmt.Printf("warning: environment variable '%s' could not interpret value '%s' as integer: %v\n", stype.Name, value, err)
				continue
			}
			if fieldval.IsNil() {
				v := reflect.ValueOf(valueAsInt)
				fieldval.Set(reflect.New(v.Type()))
			}
			fieldval.Elem().SetInt(int64(valueAsInt))

		case "*float32":
			valueAsFloat, err := strconv.ParseFloat(value, 64)
			if err != nil {
				fmt.Printf("warning: environment variable '%s' could not interpret value '%s' as float: %v\n", stype.Name, value, err)
				continue
			}
			if fieldval.IsNil() {
				v := reflect.ValueOf(valueAsFloat)
				fieldval.Set(reflect.New(v.Type()))
			}
			fieldval.Elem().SetFloat(valueAsFloat)
		default:
			panic(fmt.Sprintf("unknown kind '%s'", stype.Type.String()))
		}
	}
}
