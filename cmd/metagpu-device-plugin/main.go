package main

import (
	"fmt"
	"github.com/AccessibleAI/cnvrg-fractional-accelerator-device-plugin/pkg/deviceplugin"
	"github.com/AccessibleAI/cnvrg-fractional-accelerator-device-plugin/pkg/metagpusrv"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

type param struct {
	name      string
	shorthand string
	value     interface{}
	usage     string
	required  bool
}

var (
	Version    string
	Build      string
	rootParams = []param{
		{name: "config", shorthand: "c", value: ".", usage: "path to configuration file"},
		{name: "json-log", shorthand: "", value: false, usage: "output logs in json format"},
		{name: "verbose", shorthand: "", value: false, usage: "enable verbose logs"},
		{name: "metagpu-server-addr", shorthand: "", value: "0.0.0.0:50052", usage: "listening address for metagpu server "},
	}
	metaGpuRecalc = make(chan bool)
)

var metaGpuDevicePluginVersion = &cobra.Command{
	Use:   "version",
	Short: "Print metagpu-device-plugin version and build sha",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🐾 version: %s build: %s \n", Version, Build)
	},
}

var metaGpuStart = &cobra.Command{
	Use:   "start",
	Short: "Start metagpu device plugin",
	Run: func(cmd *cobra.Command, args []string) {
		f := deviceplugin.NewMetaGpuDevicePlugin(metaGpuRecalc)
		f.Start()
		metagpusrv.NewMetaGpuServer(f).Start()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		for {
			select {
			case s := <-sigCh:
				log.Infof("signal: %s, shutting down", s)
				f.Stop()
				log.Info("bye bye 👋")
				os.Exit(0)

			}
		}
	},
}

var rootCmd = &cobra.Command{
	Use:   "metagpu",
	Short: "Metagpu - fractional accelerator device plugin",
}

func init() {
	cobra.OnInitialize(initConfig)
	setParams(rootParams, rootCmd)
	rootCmd.AddCommand(metaGpuDevicePluginVersion)
	rootCmd.AddCommand(metaGpuStart)

}

func initConfig() {
	viper.AutomaticEnv()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(viper.GetString("config"))
	viper.SetEnvPrefix("METAGPU_DEVICE_PLUGIN")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	setupLogging()
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("config file not found, err: %s", err)
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Infof("config file changed: %s, triggering meta gpu recalculation", e.Name)
		metaGpuRecalc <- true
	})
}

func setParams(params []param, command *cobra.Command) {
	for _, param := range params {
		switch v := param.value.(type) {
		case int:
			command.PersistentFlags().IntP(param.name, param.shorthand, v, param.usage)
		case string:
			command.PersistentFlags().StringP(param.name, param.shorthand, v, param.usage)
		case bool:
			command.PersistentFlags().BoolP(param.name, param.shorthand, v, param.usage)
		}
		if err := viper.BindPFlag(param.name, command.PersistentFlags().Lookup(param.name)); err != nil {
			panic(err)
		}
	}
}

func setupLogging() {

	// Set log verbosity
	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// Set log format
	if viper.GetBool("json-log") {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetReportCaller(true)
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
			CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
				fileName := fmt.Sprintf(" [%s]", path.Base(frame.Function)+":"+strconv.Itoa(frame.Line))
				return "", fileName
			},
		})
	}

	// Logs are always goes to STDOUT
	log.SetOutput(os.Stdout)
}

func main() {

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
