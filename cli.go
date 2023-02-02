package main

import (
	"fmt"
	"os"
	"strings"
	"traductio/internal/inputreader"
	"traductio/internal/sink"
	_ "traductio/internal/sink/influx"
	_ "traductio/internal/sink/timestream"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type App struct {
	cfgFile string

	cfg struct {
		vars []string
		run  struct {
			stopAfter string
		}
	}

	// entry point
	Execute func() error
}

func NewApp() *App {
	a := &App{}
	appName := "traductio"

	// root
	rootCmd := &cobra.Command{
		Use:   appName,
		Short: "Read data from JSON, process it and store the results in a time series database",
	}
	rootCmd.PersistentFlags().StringSliceVarP(&a.cfg.vars, "vars", "v", []string{}, "key:value pairs of variables to be used in the input templates")
	rootCmd.PersistentFlags().StringVarP(&a.cfgFile, "cfg", "c", fmt.Sprintf("$HOME/%s.yaml", appName), "configuration file path")
	a.Execute = rootCmd.Execute

	// config
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Performs all steps",
		Long:  ``,
		Run:   a.runCmd,
	}
	runCmd.PersistentFlags().StringVar(&a.cfg.run.stopAfter, "stop-after", "", fmt.Sprintf("name of the step to stop afterwards, can be one of: %s", strings.Join(GetSteps(), ", ")))
	rootCmd.AddCommand(runCmd)

	// version
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version info",
		Run:   a.versionCmd,
	}
	rootCmd.AddCommand(versionCmd)

	return a
}

func (a *App) runCmd(cmd *cobra.Command, args []string) {
	fmt.Printf("Run command: cfg=%s\n", a.cfgFile)
	// validating the 'stopAfter' flag
	if a.cfg.run.stopAfter != "" {
		found := false
		for _, stepName := range GetSteps() {
			if stepName == a.cfg.run.stopAfter {
				found = true
				break
			}
		}
		if found {
			info(fmt.Sprintf("Running until step '%s'", a.cfg.run.stopAfter))
		} else {
			steps := strings.Join(GetSteps(), ", ")
			err := fmt.Errorf("there is no step called '%s', should be one of the following: %s", a.cfg.run.stopAfter, steps)
			exitOnErr(err)
		}
	}

	// STEP ReadConfig
	c, err := ReadConfig(a.cfgFile)
	exitOnErr(err)

	if a.cfg.run.stopAfter == StepReadConfig.String() {
		info("Printing configuration file as read to STDOUT and exiting...")
		fmt.Println(c)
		return
	}

	// STEP PreFetch
	vars, err := sliceToMap(a.cfg.vars, ":")
	exitOnErr(err)

	i, err := inputreader.NewInput(c.Input, vars)
	exitOnErr(err)

	if a.cfg.run.stopAfter == StepPreFetch.String() {
		info("Printing rendered input data to STDOUT and exiting...")
		b, _ := yaml.Marshal(i)
		fmt.Println(string(b))
		return
	}

	// STEP Fetch
	data, err := i.Fetch()
	exitOnErr(err)

	if a.cfg.run.stopAfter == StepFetch.String() {
		info("Printing fetched data to STDOUT and exiting...")
		fmt.Println(string(data))
		return
	}

	// STEP Validate
	_, errs := c.Validators.ValidateContent(data)
	exitOnErr(errs...)

	if a.cfg.run.stopAfter == StepValidate.String() {
		info("Validation was successful, exiting...")
		return
	}

	// STEP Process
	//points, _, fragment, err := Process(data, c.Process.Iterator, sink.Point{}, true)
	points, _, _, err := Process(data, c.Process.Iterator, sink.Point{}, false)
	exitOnErr(err)

	if a.cfg.run.stopAfter == StepProcess.String() {
		info("Printing extracted points to STDOUT and last iterator fragment to STDERR and exiting...")

		if !c.Process.NoTrim {
			points, _, _ = sink.TrimPoints(points)
		}
		table, err := sink.PointsAsCSV(points, ",")
		exitOnErr(err)

		fmt.Println(string(table))

		//info(fragment)
		return
	}
	// STEP Store
	fmt.Println("Creating sink")
	t, err := sink.New(c.Output)
	exitOnErr(err)
	fmt.Printf("Sink for %s created\n", t.GetName())

	if len(points) < 1 {
		fmt.Println("No data points to save")
		os.Exit(0)
	}

	fmt.Printf("Saving %d data points to sink\n", len(points))
	err = t.Write(points)
	defer t.Close()
	exitOnErr(err)
	fmt.Println("Data points saved")
}

func (a *App) versionCmd(cmd *cobra.Command, args []string) {
	fmt.Println(versionInfo())
}
