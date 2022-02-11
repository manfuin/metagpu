package main

import (
	pbdevice "github.com/AccessibleAI/cnvrg-fractional-accelerator-device-plugin/gen/proto/go/device/v1"
	"github.com/jedib0t/go-pretty/v6/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"os"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list resources",
}

var processListParams = []param{
	{name: "watch", shorthand: "w", value: false, usage: "watch for the changes"},
}

var processesListCmd = &cobra.Command{
	Use:   "process",
	Short: "list gpu processes, and processes metadata",
	Run: func(cmd *cobra.Command, args []string) {
		//listDevicesProcesses()
		watchDevicesProcesses()
	},
}

func watchDevicesProcesses() {
	conn, err := GetGrpcMetaGpuSrvClientConn()
	if err != nil {
		log.Fatalf("can't initiate connection to metagpu server, %s", err)
	}
	device := pbdevice.NewDeviceServiceClient(conn)
	hostname, err := os.Hostname()
	if err != nil {
		log.Errorf("faild to detect podId, err: %s", err)
	}
	request := &pbdevice.StreamDeviceProcessesRequest{PodId: hostname}
	stream, err := device.StreamDeviceProcesses(authenticatedContext(), request)

	for {
		processes, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error watching gpu processes, err: %s", err)
		}
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		header := table.Row{
			"Device UUID",
			"Pid",
			"GpuMemory",
			"Command",
			"Pod",
			"Namespace",
			"Metagpus",
		}
		t.AppendHeader(header)
		var rows []table.Row
		for _, deviceProcess := range processes.DevicesProcesses {
			rows = append(rows, table.Row{
				deviceProcess.Uuid,
				deviceProcess.Pid,
				deviceProcess.Memory / (1024 * 1024),
				deviceProcess.Cmdline,
				deviceProcess.PodName,
				deviceProcess.PodNamespace,
				deviceProcess.MetagpuRequests,
			})
		}
		t.AppendRows(rows)
		t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		t.AppendFooter(table.Row{"", "", "Free: 57%", "", "", "", "Total: 8"})
		t.Render()
	}
}

func listDevicesProcesses() {

	conn, err := GetGrpcMetaGpuSrvClientConn()
	if err != nil {
		log.Fatalf("can't initiate connection to metagpu server, %s", err)
	}
	device := pbdevice.NewDeviceServiceClient(conn)
	hostname, err := os.Hostname()
	if err != nil {
		log.Errorf("faild to detect podId, err: %s", err)
	}
	ldr := &pbdevice.ListDeviceProcessesRequest{PodId: hostname}
	resp, err := device.ListDeviceProcesses(authenticatedContext(), ldr)
	if err != nil {
		log.Errorf("falid to list device processes, err: %s ", err)
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	header := table.Row{
		"Device UUID",
		"Pid",
		"GpuMemory",
		"Command",
		"Pod",
		"Namespace",
		"Metagpus",
	}
	t.AppendHeader(header)
	var rows []table.Row
	for _, deviceProcess := range resp.DevicesProcesses {
		rows = append(rows, table.Row{
			deviceProcess.Uuid,
			deviceProcess.Pid,
			deviceProcess.Memory / (1024 * 1024),
			deviceProcess.Cmdline,
			deviceProcess.PodName,
			deviceProcess.PodNamespace,
			deviceProcess.MetagpuRequests,
		})
	}
	t.AppendRows(rows)
	t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
	t.AppendFooter(table.Row{"", "", "Free: 57%", "", "", "", "Total: 8"})
	t.Render()
}
