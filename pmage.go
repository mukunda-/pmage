package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"go.mukunda.com/pmage/clog"
	"go.mukunda.com/pmage/pmage"
)

var VERSION = "0.1"

var usageText = strings.TrimSpace(`
Usage: pmage [options] inputpath outputpath
Use --help for more info.`)

var helpText = strings.TrimSpace(`
Usage: pmage [options] inputpath outputpath

Options:
--profile PROFILE, -p PROFILE
  Select device profile. Can be "snes".

--export TYPE, -e TYPE
  Select export type. Can be "ca65".
`)

type Config struct {
	InputFilePath  string
	OutputFilePath string
	Profile        string
	Help           bool
	ExportType     string
	Version        bool
}

func getBuildCommit() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return ""
}

func printVersion() {
	commit := getBuildCommit()
	if commit != "" {
		fmt.Println("pmage version", VERSION, commit)
	} else {
		fmt.Println("pmage version", VERSION)
	}
}

func pmageCli(args []string) int {

	flags := flag.NewFlagSet("pmage", flag.ExitOnError)

	var config Config
	flags.StringVar(&config.Profile, "export", "", "Select export type [ca65]")
	flags.StringVar(&config.Profile, "e", "", "Select export type [ca65]")
	flags.StringVar(&config.Profile, "profile", "", "Select device profile")
	flags.StringVar(&config.Profile, "p", "", "Select device profile")
	flags.BoolVar(&config.Help, "help", false, "Show help")
	flags.BoolVar(&config.Help, "h", false, "Show help")
	flags.BoolVar(&config.Help, "?", false, "Show help")
	flags.BoolVar(&config.Version, "version", false, "Show version")
	flags.Parse(args)

	if config.Help {
		fmt.Println(helpText)
		return 0
	}

	if config.Version {
		printVersion()
		return 0
	}

	config.InputFilePath = flags.Arg(0)
	config.OutputFilePath = flags.Arg(1)

	if config.InputFilePath == "" {
		clog.Errorln("No input file path specified.")
		clog.Errorln(usageText)
		return 1
	}

	if len(flags.Args()) < 2 {
		clog.Errorln("No output file path specified.")
		clog.Errorln(usageText)
		return 1
	}

	var p pmage.Profile
	config.Profile = strings.ToLower(config.Profile)
	switch config.Profile {
	case "":
		clog.Infoln("Defaulting to SNES profile.")
		p.System = "snes"
	case "snes": // Add valid profiles here.
		p.System = config.Profile
	default:
		clog.Errorf("Unknown profile: %s\n", config.Profile)
		return 1
	}

	if config.ExportType == "" {
		clog.Infoln("Defaulting to ca65 export.")
		config.ExportType = "ca65"
	}

	converter := pmage.NewConverter(&p)
	err := converter.Convert(config.InputFilePath, config.OutputFilePath, strings.ToLower(config.ExportType))
	if err != nil {
		clog.Errorln(err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(pmageCli(os.Args[1:]))
}
