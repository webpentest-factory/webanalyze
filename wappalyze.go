package webanalyze

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const WAPPALYZER_URL = "https://raw.githubusercontent.com/AliasIO/Wappalyzer/master/src/apps.json"

type StringArray []string

type App struct {
	Cats    []int             `json:"cats"`
	Headers map[string]string `json:"headers"`
	Meta    map[string]string `json:"meta"`
	HTML    StringArray       `json:"html"`
	Script  StringArray       `json:"script"`
	URL     StringArray       `json:"url"`
	Website string            `json:"website"`

	HTMLRegex   []*regexp.Regexp `json:"-"`
	ScriptRegex []*regexp.Regexp `json:"-"`
	URLRegex    []*regexp.Regexp `json:"-"`
	HeaderRegex []NamedRegexp    `json:"-"`
	MetaRegex   []NamedRegexp    `json:"-"`
}

type AppsDefinition struct {
	Apps map[string]App `json:"apps"`
}

type Match struct {
	AppName    string     `json:"app"`
	AppWebsite string     `json:"app_website"`
	Matches    [][]string `json:"matches"`
}

type NamedRegexp struct {
	Name  string
	Regex *regexp.Regexp
}

// custom unmarshaler for handling bogus apps.json types from wappalyzer
func (t *StringArray) UnmarshalJSON(data []byte) error {
	var s string
	var sa []string

	if err := json.Unmarshal(data, &s); err != nil {

		// not a string, so maybe []string?
		if err := json.Unmarshal(data, &sa); err != nil {
			return err
		}
		*t = sa
		return nil
	}
	*t = StringArray{s}
	return nil
}

func updateApps(url string) error {
	return DownloadFile(url, WAPPALYZER_URL)
}

func DownloadFile(from, to string) error {

	resp, err := http.Get(from)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, resp.Body)
	return err
}

// load apps from file
func loadApps(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(f)
	if err = dec.Decode(&appDefs); err != nil {
		return err
	}

	// compile regular expressions
	for key, value := range appDefs.Apps {

		app := appDefs.Apps[key]
		app.HTMLRegex = compileRegexes(value.HTML)
		app.ScriptRegex = compileRegexes(value.Script)
		app.URLRegex = compileRegexes(value.URL)
		app.HeaderRegex = []NamedRegexp{}

		for key, value := range app.Headers {

			if value == "" {
				continue
			}

			h := NamedRegexp{
				Name: key,
			}

			// Filter out webapplyzer attributes from regular expression
			splitted := strings.Split(value, "\\;")

			r, err := regexp.Compile(splitted[0])
			if err == nil {
				h.Regex = r
				app.HeaderRegex = append(app.HeaderRegex, h)
			}
		}

		for key, value := range app.Meta {

			if value == "" {
				continue
			}

			// Filter out webapplyzer attributes from regular expression
			splitted := strings.Split(value, "\\;")

			h := NamedRegexp{
				Name: key,
			}

			r, err := regexp.Compile(splitted[0])
			if err == nil {
				h.Regex = r
				app.MetaRegex = append(app.MetaRegex, h)
			}
		}

		appDefs.Apps[key] = app

	}

	return nil
}
func compileRegexes(s StringArray) []*regexp.Regexp {
	var list []*regexp.Regexp

	for _, regexString := range s {

		// Filter out webapplyzer attributes from regular expression
		cleaned := strings.Split(regexString, "\\;")[0]

		regex, err := regexp.Compile(cleaned)
		if err != nil {
			// ignore failed compiling for now
			// log.Printf("warning: compiling regexp for failed: %v", regexString, err)
		} else {
			list = append(list, regex)
		}
	}

	return list
}
