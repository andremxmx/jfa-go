package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Storage struct {
	timePattern                                                                                            string
	invite_path, emails_path, policy_path, configuration_path, displayprefs_path, ombi_path, profiles_path string
	invites                                                                                                Invites
	profiles                                                                                               map[string]Profile
	defaultProfile                                                                                         string
	emails, policy, configuration, displayprefs, ombi_template                                             map[string]interface{}
	lang                                                                                                   Lang
}

type EmailLang map[string]map[string]map[string]interface{} // Map of lang codes to email name to fields

func (el *EmailLang) format(lang, email, field string, vals ...string) string {
	text := (*el)[lang][email][field].(string)
	for _, val := range vals {
		text = strings.Replace(text, "{n}", val, 1)
	}
	return text
}
func (el *EmailLang) get(lang, email, field string) string { return (*el)[lang][email][field].(string) }

type Lang struct {
	chosenFormLang  string
	chosenAdminLang string
	chosenEmailLang string
	AdminPath       string
	Admin           map[string]map[string]interface{}
	AdminJSON       map[string]string
	FormPath        string
	Form            map[string]map[string]interface{}
	EmailPath       string
	Email           EmailLang
}

// timePattern: %Y-%m-%dT%H:%M:%S.%f

type Profile struct {
	Admin         bool                   `json:"admin,omitempty"`
	LibraryAccess string                 `json:"libraries,omitempty"`
	FromUser      string                 `json:"fromUser,omitempty"`
	Policy        map[string]interface{} `json:"policy,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Displayprefs  map[string]interface{} `json:"displayprefs,omitempty"`
	Default       bool                   `json:"default,omitempty"`
}

type Invite struct {
	Created       time.Time                  `json:"created"`
	NoLimit       bool                       `json:"no-limit"`
	RemainingUses int                        `json:"remaining-uses"`
	ValidTill     time.Time                  `json:"valid_till"`
	Email         string                     `json:"email"`
	UsedBy        [][]string                 `json:"used-by"`
	Notify        map[string]map[string]bool `json:"notify"`
	Profile       string                     `json:"profile"`
}

type Invites map[string]Invite

func (st *Storage) loadInvites() error {
	return loadJSON(st.invite_path, &st.invites)
}

func (st *Storage) storeInvites() error {
	return storeJSON(st.invite_path, st.invites)
}

func (st *Storage) loadLang() error {
	loadData := func(path string, stringJson bool) (map[string]string, map[string]map[string]interface{}, error) {
		files, err := ioutil.ReadDir(path)
		outString := map[string]string{}
		out := map[string]map[string]interface{}{}
		if err != nil {
			return nil, nil, err
		}
		for _, f := range files {
			index := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
			var data map[string]interface{}
			var file []byte
			var err error
			file, err = ioutil.ReadFile(filepath.Join(path, f.Name()))
			if err != nil {
				file = []byte("{}")
			}
			// Replace Jellyfin with something if necessary
			if substituteStrings != "" {
				fileString := strings.ReplaceAll(string(file), "Jellyfin", substituteStrings)
				file = []byte(fileString)
			}
			err = json.Unmarshal(file, &data)
			if err != nil {
				log.Printf("ERROR: Failed to read \"%s\": %s", path, err)
				return nil, nil, err
			}
			if stringJson {
				stringJSON, err := json.Marshal(data)
				if err != nil {
					return nil, nil, err
				}
				outString[index] = string(stringJSON)
			}
			out[index] = data

		}
		return outString, out, nil
	}
	_, form, err := loadData(st.lang.FormPath, false)
	if err != nil {
		return err
	}
	for index, lang := range form {
		strings := lang["strings"].(map[string]interface{})
		validationStrings := strings["validationStrings"].(map[string]interface{})
		vS, err := json.Marshal(validationStrings)
		if err != nil {
			return err
		}
		strings["validationStrings"] = string(vS)
		lang["strings"] = strings
		form[index] = lang
	}
	st.lang.Form = form
	adminJSON, admin, err := loadData(st.lang.AdminPath, true)
	st.lang.Admin = admin
	st.lang.AdminJSON = adminJSON

	_, emails, err := loadData(st.lang.EmailPath, false)
	fixedEmails := map[string]map[string]map[string]interface{}{}
	for lang, e := range emails {
		f := map[string]map[string]interface{}{}
		for field, vals := range e {
			f[field] = vals.(map[string]interface{})
		}
		fixedEmails[lang] = f
	}
	st.lang.Email = fixedEmails
	return err
}

func (st *Storage) loadEmails() error {
	return loadJSON(st.emails_path, &st.emails)
}

func (st *Storage) storeEmails() error {
	return storeJSON(st.emails_path, st.emails)
}

func (st *Storage) loadPolicy() error {
	return loadJSON(st.policy_path, &st.policy)
}

func (st *Storage) storePolicy() error {
	return storeJSON(st.policy_path, st.policy)
}

func (st *Storage) loadConfiguration() error {
	return loadJSON(st.configuration_path, &st.configuration)
}

func (st *Storage) storeConfiguration() error {
	return storeJSON(st.configuration_path, st.configuration)
}

func (st *Storage) loadDisplayprefs() error {
	return loadJSON(st.displayprefs_path, &st.displayprefs)
}

func (st *Storage) storeDisplayprefs() error {
	return storeJSON(st.displayprefs_path, st.displayprefs)
}

func (st *Storage) loadOmbiTemplate() error {
	return loadJSON(st.ombi_path, &st.ombi_template)
}

func (st *Storage) storeOmbiTemplate() error {
	return storeJSON(st.ombi_path, st.ombi_template)
}

func (st *Storage) loadProfiles() error {
	err := loadJSON(st.profiles_path, &st.profiles)
	for name, profile := range st.profiles {
		if profile.Default {
			st.defaultProfile = name
		}
		change := false
		if profile.Policy["IsAdministrator"] != nil {
			profile.Admin = profile.Policy["IsAdministrator"].(bool)
			change = true
		}
		if profile.Policy["EnabledFolders"] != nil {
			length := len(profile.Policy["EnabledFolders"].([]interface{}))
			if length == 0 {
				profile.LibraryAccess = "All"
			} else {
				profile.LibraryAccess = strconv.Itoa(length)
			}
			change = true
		}
		if profile.FromUser == "" {
			profile.FromUser = "Unknown"
			change = true
		}
		if change {
			st.profiles[name] = profile
		}
	}
	if st.defaultProfile == "" {
		for n := range st.profiles {
			st.defaultProfile = n
		}
	}
	return err
}

func (st *Storage) storeProfiles() error {
	return storeJSON(st.profiles_path, st.profiles)
}

func (st *Storage) migrateToProfile() error {
	st.loadPolicy()
	st.loadConfiguration()
	st.loadDisplayprefs()
	st.loadProfiles()
	st.profiles["Default"] = Profile{
		Policy:        st.policy,
		Configuration: st.configuration,
		Displayprefs:  st.displayprefs,
	}
	return st.storeProfiles()
}

func loadJSON(path string, obj interface{}) error {
	var file []byte
	var err error
	file, err = ioutil.ReadFile(path)
	if err != nil {
		file = []byte("{}")
	}
	err = json.Unmarshal(file, &obj)
	if err != nil {
		log.Printf("ERROR: Failed to read \"%s\": %s", path, err)
	}
	return err
}

func storeJSON(path string, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, data, 0644)
	if err != nil {
		log.Printf("ERROR: Failed to write to \"%s\": %s", path, err)
	}
	return err
}

// One build of JF 10.7.0 hyphenated user IDs while another one later didn't. These functions will hyphenate/de-hyphenate email storage.

func hyphenate(userID string) string {
	if userID[8] == '-' {
		return userID
	}
	return userID[:8] + "-" + userID[8:12] + "-" + userID[12:16] + "-" + userID[16:20] + "-" + userID[20:]
}

func (app *appContext) deHyphenateEmailStorage(old map[string]interface{}) (map[string]interface{}, int, error) {
	jfUsers, status, err := app.jf.GetUsers(false)
	if status != 200 || err != nil {
		return nil, status, err
	}
	newEmails := map[string]interface{}{}
	for _, user := range jfUsers {
		unHyphenated := user["Id"].(string)
		hyphenated := hyphenate(unHyphenated)
		email, ok := old[hyphenated]
		if ok {
			newEmails[unHyphenated] = email
		}
	}
	return newEmails, status, err
}

func (app *appContext) hyphenateEmailStorage(old map[string]interface{}) (map[string]interface{}, int, error) {
	jfUsers, status, err := app.jf.GetUsers(false)
	if status != 200 || err != nil {
		return nil, status, err
	}
	newEmails := map[string]interface{}{}
	for _, user := range jfUsers {
		unstripped := user["Id"].(string)
		stripped := strings.ReplaceAll(unstripped, "-", "")
		email, ok := old[stripped]
		if ok {
			newEmails[unstripped] = email
		}
	}
	return newEmails, status, err
}
