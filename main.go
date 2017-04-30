package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/heroku/docker-registry-client/registry"
	yaml "gopkg.in/yaml.v2"
)

func errorAndExit(e error) {
	if e != nil {
		fmt.Println(e.Error())
		os.Exit(1)
	}
}

var (
	config = flag.String("config", "config.yaml", "Configuration file location.")
)

type RegistryConfig struct {
	Name         string `yaml:"name"`
	Address      string `yaml:"address"`
	Username     string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
	PasswordFile string `yaml:"passwordFile,omitempty"`
	Email        string `yaml:"email,omitempty"`
}

type AccountList struct {
	Accounts []RegistryConfig `yaml:"accounts"`
}

type Config struct {
	AccountList AccountList `yaml:"dockerRegistry"`
}

func parseConfig(cfg []byte) ([]RegistryConfig, error) {
	parsed := Config{}

	err := yaml.Unmarshal(cfg, &parsed)
	if err != nil {
		return nil, err
	}

	return parsed.AccountList.Accounts, nil
}

var RegistryCache map[string]*registry.Registry = make(map[string]*registry.Registry, 0)

func initRegistryCache(accounts []RegistryConfig) {
	for _, account := range accounts {
		//TODO: add support for `passwordFile` accounts.
		//Those will need a new registry instance each time
		hub, err := registry.New(account.Address, account.Username, account.Password)
		errorAndExit(err)
		RegistryCache[account.Name] = hub
	}
}

func main() {

	flag.Parse()

	cfg, err := ioutil.ReadFile(*config)

	errorAndExit(err)

	accounts, err := parseConfig(cfg)
	errorAndExit(err)
	initRegistryCache(accounts)

	r := mux.NewRouter()

	//get the manifest for an image
	r.HandleFunc("/{account}/{repository:.+\\/.+}/{tag}/metadata", func(res http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		hub := RegistryCache[vars["account"]]

		manifest, err := hub.Manifest(vars["repository"], vars["tag"])
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(manifest)

	})

	//get History.V1Compatibility information for an image
	r.HandleFunc("/{account}/{repository:.+\\/.+}/{tag}/history", func(res http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		hub := RegistryCache[vars["account"]]

		manifest, err := hub.Manifest(vars["repository"], vars["tag"])
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		res.Header().Set("Content-Type", "application/json")
		if param := req.URL.Query().Get("level"); param != "" {
			index, _ := strconv.Atoi(param)                            //TODO: actually handle this error
			res.Write([]byte(manifest.History[index].V1Compatibility)) //TODO: handle invalid indexes
		} else {
			json.NewEncoder(res).Encode(manifest.History)
		}

	})

	http.ListenAndServe(":8080", r)
}
