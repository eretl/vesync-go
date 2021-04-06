package vesync

import (
	"errors"
	log "github.com/rs/zerolog/log"
	"regexp"
)

var ApiRateLimit int64 = 30
var defaultTz = "America/New_York"
var defaultEnerUpInt = 21600
var deviceClass = map[string]interface{}{
"wifi-switch-1.3": VeSyncOutlet7A{},
"ESW03-USA": VeSyncOutlet10A{},
"ESW01-EU": VeSyncOutlet10A{},
"ESW15-USA": VeSyncOutlet15A{},
"ESWL01": VeSyncWallSwitch{},
"ESWL03": VeSyncWallSwitch{},
"LV-PUR131S": VeSyncAir131{},
"ESO15-TB": VeSyncOutdoorPlug{},
"ESL100": VeSyncBulbESL100{},
"ESL100CW": VeSyncBulbESL100CW{},
"ESWD16": VeSyncDimmerSwitch{},
"Classic300S": VeSync300S{},
}

var deviceTypesDict  = map[string][]string{
"outlets": {"wifi-switch-1.3", "ESW03-USA", "ESW01-EU", "ESW15-USA", "ESO15-TB"},
"switches": {"ESWL01", "ESWL03", "ESWD16"},
"fans": {"LV-PUR131S", "Classic300S"},
"bulbs": {"ESL100", "ESL100CW"},
}


type VeSync struct {
	Username                string
	TimeZone                string
	Email                   string
	Password                string
	Token                   string
	AccountId               string
	Devices                 []string
	Enabled                 bool
	UpdateInterval          int64
	LastUpdateTs            int64
	InProcess               bool
	_energy_update_interval int
	_energy_check           bool
	_dev_list               map[string][]interface{}
	outlets                 []interface{}
	switches                []interface{}
	fans                    []interface{}
	bulbs                   []interface{}
	scales                  []interface{}
}

func NewVeSync(username string, password string, timeZone string) *VeSync {
	tz := ""
	if timeZone != "" {
		validTZ := regexp.MustCompile(`[^a-zA-Z/_]`)
		if validTZ.MatchString(timeZone) {
			tz = defaultTz
			log.Debug().
				Msgf("Invalid characters in time zone - %s", timeZone)
		} else{
			tz = timeZone
		}
	} else {
		tz = defaultTz
		log.Debug().
			Msg("Time zone is empty")
	}
	return &VeSync{
		Username: username,
		Password: password,
		TimeZone: tz,
		//Token: nil,
		//AccountId: nil,
		Devices: nil,
		Enabled: false,
		UpdateInterval: ApiRateLimit,
		//LastUpdateTs: nil,
		InProcess: false,
		_energy_update_interval: defaultEnerUpInt,
		_energy_check: true,
		//_dev_list: {for k,v := range deviceTypesDict{}},
		//outlets: []string{},

	}
}

func (manager VeSync) Login() (bool, error) {
	var response map[string]interface{}

	if manager.Username == ""{
		log.Error().Msg("Username invalid")
		return false, errors.New("username invalid")
	}
	if manager.Password == "" {
		log.Error().Msg("Password invalid")
		return false, errors.New("password invalid")
	}

	_, err := CallApi("/cloud/v1/user/login", "post", nil, ReqBody(manager, "login"), &response)

	if result, ok := response["result"]; err == nil && ok && CheckCode(response) {
		manager.Token = result.(map[string]interface{})["token"].(string)
		manager.AccountId = result.(map[string]interface{})["accountID"].(string)
		manager.Enabled = true

		return true, err
	}
	return false, err
}
