package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"text/template"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/koshoi/selektor/config"
	"github.com/koshoi/selektor/db"
	"github.com/koshoi/selektor/db/options"
	"github.com/koshoi/selektor/flag"
	"github.com/koshoi/selektor/timerange"
)

var cfgPath = ""
var env = ""
var debug = false
var needHelp = false
var showQuery = false
var showTemplate = false

type flagWithConfig struct {
	*pflag.Flag
	config.FlagConfig
}

func addCommonFlags(pfs *pflag.FlagSet) {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("Failed to get user's home directory: %s", err.Error()))
	}

	pfs.StringVarP(&cfgPath, "config", "c", path.Join(home, ".config", "selektor", "config.toml"), "set path to your selektor config file")
	pfs.BoolVarP(&debug, "debug", "d", false, "enable debug output")
	pfs.StringVarP(&env, "env", "e", "", "set env to use")
	pfs.BoolVar(&showQuery, "showquery", false, "shows you raw query, that would have been sent to your database")
	pfs.BoolVar(&showTemplate, "showtemplate", false, "shows you the content of template, that would be used to compose your query")

	// specificly defining help flag to continue parsing flags when --help flag is occuring
	// to be honest it's either I do not unserstand pflag package
	// or pflag is just a huge mess
	// cobra was not made for dynamically generated commands
	pfs.BoolVarP(&needHelp, "help", "h", false, "print help message")
}

func buildSelectorCommand(sName string, cfg *config.SelektorConfig) (*cobra.Command, error) {
	sConfig := cfg.Selectors[sName]
	//flags := make(map[string]*pflag.Flag)
	flags := make(map[string]flagWithConfig)

	cmd := &cobra.Command{
		Use:   sName,
		Short: sConfig.Description,
		RunE: func(cmd *cobra.Command, args []string) error {
			if env == "" {
				for k, eConfig := range cfg.Envs {
					if eConfig.IsDefault {
						env = k
						break
					}
				}
			}

			eConfig, ok := cfg.Envs[env]
			if !ok {
				fmt.Printf("env=%s not found, available envs are:\n", env)
				for k := range cfg.Envs {
					fmt.Printf("- %s\n", k)
				}
				fmt.Println("")
				return fmt.Errorf("unknown env='%s'", env)
			}

			templateContex := make(map[string]string)
			for fName, fWithConfig := range flags {
				isDefined := fWithConfig.Changed
				if sConfig.Flags[fName].Required && !isDefined {
					fmt.Printf("--%s flag is required, but was not set\n", fName)
					return fmt.Errorf("flag='%s' is required", fName)
				}

				isSet := isDefined
				if !isSet && fWithConfig.Default != nil {
					isSet = true
				}

				fValue := ""
				if isDefined {
					fValue = fWithConfig.Value.String()
				} else {
					if isSet {
						fValue = *fWithConfig.Default
					}
				}

				if sConfig.Flags[fName].Type == config.FlagTimerange {
					tr, err := timerange.ParseRange(fValue)
					if err != nil {
						return fmt.Errorf("failed to parse value='%s' as timerange: %w", fValue, err)
					}

					from, to := tr.From, tr.To
					if sConfig.UseUTC {
						from, to = from.UTC(), to.UTC()
					}

					templateContex[fName+"From"] = from.Format("2006-01-02 15:04:05")
					templateContex[fName+"To"] = to.Format("2006-01-02 15:04:05")
				} else {
					templateContex[fName] = fValue
				}

				// __defined_flagName is used in templates to comment out lines that can contain undefined filter flags
				if isDefined {
					templateContex[fmt.Sprintf("__defined_%s", fName)] = ""
					templateContex[fmt.Sprintf("__undefined_%s", fName)] = "--"
				} else {
					templateContex[fmt.Sprintf("__defined_%s", fName)] = "--"
					templateContex[fmt.Sprintf("__undefined_%s", fName)] = ""
				}

				// __set_flagName is used in the same way
				if isSet {
					templateContex[fmt.Sprintf("__set_%s", fName)] = ""
					templateContex[fmt.Sprintf("__unset_%s", fName)] = "--"
				} else {
					templateContex[fmt.Sprintf("__set_%s", fName)] = "--"
					templateContex[fmt.Sprintf("__unset_%s", fName)] = ""
				}
			}

			fMap := template.FuncMap{}

			t, err := template.New("query").Option("missingkey=error").Funcs(fMap).Parse(sConfig.Query)
			if err != nil {
				return fmt.Errorf("invalid query template given: %s", err.Error())
			}

			if showTemplate {
				// by default json.Marshal escapes < and > signs wich looks ugly
				buffer := &bytes.Buffer{}
				encoder := json.NewEncoder(buffer)
				encoder.SetEscapeHTML(false)
				encoder.SetIndent("", "    ")
				err = encoder.Encode(templateContex)
				if err != nil {
					return fmt.Errorf("failed to show template: %s", err.Error())
				}

				fmt.Println(buffer.String())
				return nil
			}

			var buf bytes.Buffer
			err = t.Execute(&buf, templateContex)
			if err != nil {
				return fmt.Errorf("failed to template query: %s", err.Error())
			}

			if showQuery {
				fmt.Println(buf.String())
			} else {
				client, err := db.NewQueriable(eConfig)
				if err != nil {
					return fmt.Errorf("client creation failed: %s", err)
				}

				reader, err := client.Query(cmd.Context(), buf.String(), options.Options{Debug: debug})
				if err != nil {
					fmt.Printf("Query failed: %s\n", err.Error())
					os.Exit(1)
				}
				io.Copy(os.Stdout, reader)
			}

			return nil
		},
	}

	for fName, fConfig := range sConfig.Flags {
		defValue := ""
		if fConfig.Default != nil {
			defValue = *fConfig.Default
		}

		pf := &pflag.Flag{
			Name:     fName,
			Usage:    fConfig.Description,
			DefValue: defValue,
			Value:    &flag.FlagValue{FlagType: "string"},
		}
		flags[fName] = flagWithConfig{pf, fConfig}
		cmd.Flags().AddFlag(pf)
	}

	return cmd, nil
}

func main() {
	fSet := pflag.NewFlagSet("basic", pflag.ContinueOnError)
	fSet.ParseErrorsWhitelist.UnknownFlags = true
	fSet.Usage = func() {}

	addCommonFlags(fSet)

	err := fSet.Parse(os.Args)
	if err != nil {
		fmt.Printf("Error occured while parsing basic flags: %s\n", err.Error())
		os.Exit(1)
	}

	cfg, err := config.ReadConfig(cfgPath)
	if err != nil {
		fmt.Printf("Failed to read config: %s\n", err.Error())
		os.Exit(1)
	}

	var rc = &cobra.Command{
		Use:   "selektor selectorName [FLAGS]",
		Short: "selektor is a tool to template and execute SQL queries",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		SilenceUsage: true,
	}
	addCommonFlags(rc.Flags())

	for sName := range cfg.Selectors {
		cmd, err := buildSelectorCommand(sName, cfg)
		if err != nil {
			fmt.Printf("Can't build selector %s: %s\n", sName, err.Error())
			os.Exit(1)
		}
		addCommonFlags(cmd.Flags())
		rc.AddCommand(cmd)
	}

	if err := rc.Execute(); err != nil {
		os.Exit(1)
	}
}
