package kosmixutil

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
)

type SystemInfoOut struct {
	Version string `json:"version"`
	System  struct {
		Manufacturer string `json:"manufacturer"`
		Model        string `json:"model"`
		Version      string `json:"version"`
		Serial       string `json:"serial"`
		UUID         string `json:"uuid"`
		Sku          string `json:"sku"`
		Virtual      bool   `json:"virtual"`
	} `json:"system"`
	Bios struct {
		Vendor      string   `json:"vendor"`
		Version     string   `json:"version"`
		ReleaseDate string   `json:"releaseDate"`
		Revision    string   `json:"revision"`
		Serial      string   `json:"serial"`
		Features    []string `json:"features"`
	} `json:"bios"`
	Baseboard struct {
		Manufacturer string `json:"manufacturer"`
		Model        string `json:"model"`
		Version      string `json:"version"`
		Serial       string `json:"serial"`
		AssetTag     string `json:"assetTag"`
		MemMax       int64  `json:"memMax"`
		MemSlots     int    `json:"memSlots"`
	} `json:"baseboard"`
	Chassis struct {
		Manufacturer string `json:"manufacturer"`
		Model        string `json:"model"`
		Type         string `json:"type"`
		Version      string `json:"version"`
		Serial       string `json:"serial"`
		AssetTag     string `json:"assetTag"`
		Sku          string `json:"sku"`
	} `json:"chassis"`
	Os struct {
		Platform    string `json:"platform"`
		Distro      string `json:"distro"`
		Release     string `json:"release"`
		Codename    string `json:"codename"`
		Kernel      string `json:"kernel"`
		Arch        string `json:"arch"`
		Hostname    string `json:"hostname"`
		Fqdn        string `json:"fqdn"`
		Codepage    string `json:"codepage"`
		Logofile    string `json:"logofile"`
		Serial      string `json:"serial"`
		Build       string `json:"build"`
		Servicepack string `json:"servicepack"`
		Uefi        bool   `json:"uefi"`
	} `json:"os"`
	UUID struct {
		Os       string   `json:"os"`
		Hardware string   `json:"hardware"`
		Macs     []string `json:"macs"`
	} `json:"uuid"`
	// Versions struct {
	// 	Kernel           string `json:"kernel"`
	// 	Openssl          string `json:"openssl"`
	// 	SystemOpenssl    string `json:"systemOpenssl"`
	// 	SystemOpensslLib string `json:"systemOpensslLib"`
	// 	Node             string `json:"node"`
	// 	V8               string `json:"v8"`
	// 	Npm              string `json:"npm"`
	// 	Yarn             string `json:"yarn"`
	// 	Pm2              string `json:"pm2"`
	// 	Gulp             string `json:"gulp"`
	// 	Grunt            string `json:"grunt"`
	// 	Git              string `json:"git"`
	// 	Tsc              string `json:"tsc"`
	// 	Mysql            string `json:"mysql"`
	// 	Redis            string `json:"redis"`
	// 	Mongodb          string `json:"mongodb"`
	// 	Apache           string `json:"apache"`
	// 	Nginx            string `json:"nginx"`
	// 	Php              string `json:"php"`
	// 	Docker           string `json:"docker"`
	// 	Postfix          string `json:"postfix"`
	// 	Postgresql       string `json:"postgresql"`
	// 	Perl             string `json:"perl"`
	// 	Python           string `json:"python"`
	// 	Python3          string `json:"python3"`
	// 	Pip              string `json:"pip"`
	// 	Pip3             string `json:"pip3"`
	// 	Java             string `json:"java"`
	// 	Gcc              string `json:"gcc"`
	// 	Virtualbox       string `json:"virtualbox"`
	// 	Bash             string `json:"bash"`
	// 	Zsh              string `json:"zsh"`
	// 	Fish             string `json:"fish"`
	// 	Powershell       string `json:"powershell"`
	// 	Dotnet           string `json:"dotnet"`
	// } `json:"versions"`
	CPU struct {
		Manufacturer     string  `json:"manufacturer"`
		Brand            string  `json:"brand"`
		Vendor           string  `json:"vendor"`
		Family           string  `json:"family"`
		Model            string  `json:"model"`
		Stepping         string  `json:"stepping"`
		Revision         string  `json:"revision"`
		Voltage          string  `json:"voltage"`
		Speed            float64 `json:"speed"`
		SpeedMin         any     `json:"speedMin"`
		SpeedMax         any     `json:"speedMax"`
		Governor         string  `json:"governor"`
		Cores            int     `json:"cores"`
		PhysicalCores    int     `json:"physicalCores"`
		PerformanceCores int     `json:"performanceCores"`
		EfficiencyCores  int     `json:"efficiencyCores"`
		Processors       int     `json:"processors"`
		Socket           string  `json:"socket"`
		Flags            string  `json:"flags"`
		Virtualization   bool    `json:"virtualization"`
		Cache            struct {
			L1D int `json:"l1d"`
			L1I int `json:"l1i"`
			L2  int `json:"l2"`
			L3  int `json:"l3"`
		} `json:"cache"`
	} `json:"cpu"`
	Graphics struct {
		Controllers []struct {
			Vendor      string `json:"vendor"`
			SubVendor   string `json:"subVendor"`
			Model       string `json:"model"`
			Bus         string `json:"bus"`
			BusAddress  string `json:"busAddress"`
			Vram        int    `json:"vram"`
			VramDynamic bool   `json:"vramDynamic"`
			PciID       string `json:"pciID"`
		} `json:"controllers"`
		Displays []any `json:"displays"`
	} `json:"graphics"`
	// Net []struct {
	// 	Iface          string  `json:"iface"`
	// 	IfaceName      string  `json:"ifaceName"`
	// 	Default        bool    `json:"default"`
	// 	IP4            string  `json:"ip4"`
	// 	IP4Subnet      string  `json:"ip4subnet"`
	// 	IP6            string  `json:"ip6"`
	// 	IP6Subnet      string  `json:"ip6subnet"`
	// 	Mac            string  `json:"mac"`
	// 	Internal       bool    `json:"internal"`
	// 	Virtual        bool    `json:"virtual"`
	// 	Operstate      string  `json:"operstate"`
	// 	Type           string  `json:"type"`
	// 	Duplex         string  `json:"duplex"`
	// 	Mtu            float64 `json:"mtu"`
	// 	Speed          any     `json:"speed"`
	// 	Dhcp           bool    `json:"dhcp"`
	// 	DNSSuffix      string  `json:"dnsSuffix"`
	// 	Ieee8021XAuth  string  `json:"ieee8021xAuth"`
	// 	Ieee8021XState string  `json:"ieee8021xState"`
	// 	CarrierChanges int     `json:"carrierChanges"`
	// } `json:"net"`
	MemLayout []struct {
		Size              int64  `json:"size"`
		Bank              string `json:"bank"`
		Type              string `json:"type"`
		Ecc               bool   `json:"ecc"`
		ClockSpeed        any    `json:"clockSpeed"`
		FormFactor        string `json:"formFactor"`
		Manufacturer      string `json:"manufacturer"`
		PartNum           string `json:"partNum"`
		SerialNum         string `json:"serialNum"`
		VoltageConfigured any    `json:"voltageConfigured"`
		VoltageMin        any    `json:"voltageMin"`
		VoltageMax        any    `json:"voltageMax"`
	} `json:"memLayout"`
	// DiskLayout []struct {
	// 	Device            string `json:"device"`
	// 	Type              string `json:"type"`
	// 	Name              string `json:"name"`
	// 	Vendor            string `json:"vendor"`
	// 	Size              int64  `json:"size"`
	// 	BytesPerSector    any    `json:"bytesPerSector"`
	// 	TotalCylinders    any    `json:"totalCylinders"`
	// 	TotalHeads        any    `json:"totalHeads"`
	// 	TotalSectors      any    `json:"totalSectors"`
	// 	TotalTracks       any    `json:"totalTracks"`
	// 	TracksPerCylinder any    `json:"tracksPerCylinder"`
	// 	SectorsPerTrack   any    `json:"sectorsPerTrack"`
	// 	FirmwareRevision  string `json:"firmwareRevision"`
	// 	SerialNum         string `json:"serialNum"`
	// 	InterfaceType     string `json:"interfaceType"`
	// 	SmartStatus       string `json:"smartStatus"`
	// 	Temperature       any    `json:"temperature"`
	// } `json:"diskLayout"`
}

func GetWrapperPath() string {
	if runtime.GOOS == "windows" {
		return "wrapper.exe"
	}
	return "wrapper"
}

func GetSystemInfo() (*SystemInfoOut, error) {
	cmd := exec.Command("./"+GetWrapperPath(), "pc-info")
	cmd.Dir, _ = os.Getwd()
	out, err := cmd.Output()
	if err != nil {
		// panic(err)
		return nil, err
	}
	var sysInfo SystemInfoOut
	err = json.Unmarshal(out, &sysInfo)
	if err != nil {
		// panic(err)
		return nil, err
	}
	return &sysInfo, nil
}

var SubstribeDynamic DynamicData = DynamicData{}

func GetDynamicData() {
	cmd := exec.Command("./wrapper", "network")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			data := scanner.Text()
			// fmt.Println("Pointer refreshed")
			err = json.Unmarshal([]byte(data), &SubstribeDynamic)
		}
	}()
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	err = cmd.Wait()
	if err != nil {
		panic(err)
	}
}

type DynamicData struct {
	FsSize []struct {
		Fs        string  `json:"fs"`
		Type      string  `json:"type"`
		Size      int64   `json:"size"`
		Used      int64   `json:"used"`
		Available int64   `json:"available"`
		Use       float64 `json:"use"`
		Mount     string  `json:"mount"`
		Rw        bool    `json:"rw"`
	} `json:"fsSize"`
	Mem struct {
		Total     int64 `json:"total"`
		Free      int64 `json:"free"`
		Used      int64 `json:"used"`
		Active    int64 `json:"active"`
		Available int64 `json:"available"`
		Buffers   int   `json:"buffers"`
		Cached    int   `json:"cached"`
		Slab      int   `json:"slab"`
		Buffcache int   `json:"buffcache"`
		Swaptotal int64 `json:"swaptotal"`
		Swapused  int   `json:"swapused"`
		Swapfree  int64 `json:"swapfree"`
		Writeback any   `json:"writeback"`
		Dirty     any   `json:"dirty"`
	} `json:"mem"`
	Network []struct {
		Iface     string `json:"iface"`
		Operstate string `json:"operstate"`
		RxBytes   int    `json:"rx_bytes"`
		RxDropped int    `json:"rx_dropped"`
		RxErrors  int    `json:"rx_errors"`
		TxBytes   int    `json:"tx_bytes"`
		TxDropped int    `json:"tx_dropped"`
		TxErrors  int    `json:"tx_errors"`
		RxSec     any    `json:"rx_sec"`
		TxSec     any    `json:"tx_sec"`
		Ms        int    `json:"ms"`
	} `json:"network"`
	CurrentLoad struct {
		AvgLoad              int     `json:"avgLoad"`
		CurrentLoad          float64 `json:"currentLoad"`
		CurrentLoadUser      float64 `json:"currentLoadUser"`
		CurrentLoadSystem    float64 `json:"currentLoadSystem"`
		CurrentLoadNice      int     `json:"currentLoadNice"`
		CurrentLoadIdle      float64 `json:"currentLoadIdle"`
		CurrentLoadIrq       float64 `json:"currentLoadIrq"`
		CurrentLoadSteal     int     `json:"currentLoadSteal"`
		CurrentLoadGuest     int     `json:"currentLoadGuest"`
		RawCurrentLoad       int     `json:"rawCurrentLoad"`
		RawCurrentLoadUser   int     `json:"rawCurrentLoadUser"`
		RawCurrentLoadSystem int     `json:"rawCurrentLoadSystem"`
		RawCurrentLoadNice   int     `json:"rawCurrentLoadNice"`
		RawCurrentLoadIdle   int     `json:"rawCurrentLoadIdle"`
		RawCurrentLoadIrq    int     `json:"rawCurrentLoadIrq"`
		RawCurrentLoadSteal  int     `json:"rawCurrentLoadSteal"`
		RawCurrentLoadGuest  int     `json:"rawCurrentLoadGuest"`
		Cpus                 []struct {
			Load          float64 `json:"load"`
			LoadUser      float64 `json:"loadUser"`
			LoadSystem    float64 `json:"loadSystem"`
			LoadNice      int     `json:"loadNice"`
			LoadIdle      float64 `json:"loadIdle"`
			LoadIrq       float64 `json:"loadIrq"`
			LoadSteal     int     `json:"loadSteal"`
			LoadGuest     int     `json:"loadGuest"`
			RawLoad       int     `json:"rawLoad"`
			RawLoadUser   int     `json:"rawLoadUser"`
			RawLoadSystem int     `json:"rawLoadSystem"`
			RawLoadNice   int     `json:"rawLoadNice"`
			RawLoadIdle   int     `json:"rawLoadIdle"`
			RawLoadIrq    int     `json:"rawLoadIrq"`
			RawLoadSteal  int     `json:"rawLoadSteal"`
			RawLoadGuest  int     `json:"rawLoadGuest"`
		} `json:"cpus"`
	} `json:"currentLoad"`
	Time struct {
		Current      int64   `json:"current"`
		Uptime       float64 `json:"uptime"`
		Timezone     string  `json:"timezone"`
		TimezoneName string  `json:"timezoneName"`
	} `json:"time"`
}
