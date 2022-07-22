package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
)

type Range struct {
	Source  string
	Service string
	Region  string
	Prefix  *net.IPNet
}

type RangeEntry struct {
	InputIP string
	Range   *Range
}

func ip2uint32(ip net.IP) uint32 {
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip)
}

func lastAddr(n *net.IPNet) (net.IP, error) { // works when the n is a prefix, otherwise...
	if n.IP.To4() == nil {
		return net.IP{}, errors.New("does not support IPv6 addresses.")
	}
	ip := make(net.IP, len(n.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(n.IP.To4())|^binary.BigEndian.Uint32(net.IP(n.Mask).To4()))
	return ip, nil
}

// regionMap is a mapping of a cloud-specific region, and a non-cloud-specific
// region.
var regionMap = map[string]string{
	// Azure mappings
	"westeurope":         "eu-west",
	"eastus":             "us-east",
	"northeurope":        "eu-north",
	"centralus":          "us-central",
	"southcentralus":     "us-southcentral",
	"westus":             "us-west",
	"southeastasia":      "ap-southeast",
	"eastasia":           "ap-east",
	"northcentralus":     "us-northcentral",
	"uksouth":            "eu-west",
	"australiaeast":      "ap-southeast",
	"canadacentral":      "ca-central",
	"japaneast":          "ap-northeast",
	"eastus2euap":        "us-east", // "Early Updates Access Program" "us-east"
	"centralindia":       "ap-south",
	"australiasoutheast": "ap-southeast",
	"brazilsouth":        "sa-east",
	"centralfrance":      "eu-west",
	"westcentralus":      "us-westcentral",
	"ukwest":             "eu-west",
	"koreacentral":       "ap-northeast",
	"japanwest":          "ap-northeast",
	"southafricanorth":   "af-south",
	"westus3":            "us-west",
	"germanywc":          "eu-central",
	"centraluseuap":      "us-central", // "Early Updates Access Program" "us-central"
	"canadaeast":         "ca-east",
	"uaenorth":           "me-south",
	"southindia":         "ap-south",
	"switzerlandn":       "eu-central",
	"norwaye":            "eu-north",
	"swedencentral":      "eu-north", // this seems correct
	"koreasouth":         "ap-northeast",
	"westindia":          "ap-south",
	"uaecentral":         "me-south", // changed based on qatar, jerusalem, abudhabi
	"southfrance":        "eu-west",
	"southafricawest":    "af-southwest",
	"switzerlandw":       "eu-central",
	"australiacentral":   "ap-southeast",
	"germanyn":           "eu-central",
	"usstagee":           "us-east", // staging for US east region? https://docs.microsoft.com/en-us/azure/virtual-network/service-tags-overview
	"brazilse":           "sa-southeast",
	"qatarcentral":       "me-south", // seems correct based on AWS region (Bahrain) + jerusalem, abudhabi
	"swedensouth":        "eu-north",
	"norwayw":            "eu-north",
	"australiacentral2":  "ap-southeast",
	"jioindiawest":       "ap-south",
	"jioindiacentral":    "ap-south",
	"israelcentral":      "me-south", // seems correct based on AWS region (Bahrain); also think this was misspelled previously
	"polandcentral":      "eu-east",
	"usstagec":           "us-central", // staging for US central region? https://docs.microsoft.com/en-us/azure/virtual-network/service-tags-overview
	"uksouth2":           "eu-west",
	"eastusslv":          "us-east", // suggestion - US east region? https://docs.microsoft.com/en-us/dotnet/api/microsoft.azure.documents.locationnames.eastusslv?view=azure-dotnet
	"uknorth":            "eu-west",
	"taiwannorth":        "ap-northeast",
	"taiwannorthwest":    "ap-northeast",
	"austriaeast":        "ap-southeast",
	"spaincentral":       "eu-south", // suggestion
	"newzealandnorth":    "ap-southeast",
	"mexicocentral":      "sa-north",
	"italynorth":         "eu-south",
	"northeurope2":       "eu-north",
	"malaysiawest":       "ap-west",
	"indiasouthcentral":  "ap-south",
	"chilec":             "sa-west", // suggestion
	"belgiumcentral":     "eu-central",
	"easteurope":         "eu-east",
	"brazilne":           "sa-northeast",
	// Google Mappings
	"us-central1":             "us-central",
	"europe-west1":            "eu-west",
	"us-east1":                "us-east",
	"asia-east1":              "ap-east",
	"us-east4":                "us-east",
	"global":                  "global",
	"asia-southeast1":         "ap-southeast",
	"us-west1":                "us-west",
	"asia-northeast1":         "ap-northeast",
	"europe-west2":            "eu-west",
	"europe-west3":            "eu-west",
	"australia-southeast1":    "ap-southeast",
	"asia-northeast3":         "ap-northeast",
	"southamerica-east1":      "sa-east",
	"europe-west4":            "eu-west",
	"us-west2":                "us-west",
	"asia-south1":             "ap-south",
	"asia-east2":              "ap-east",
	"northamerica-northeast1": "ca-east",
	"us-central2":             "us-central",
	"europe-north1":           "eu-north",
	"europe-west6":            "eu-west",
	"asia-southeast2":         "ap-southeast",
	"asia-northeast2":         "ap-northeast",
	"us-west3":                "us-west",
	"us-west4":                "us-west",
	"northamerica-northeast2": "ca-east",
	"europe-west9":            "eu-west",
	"europe-central2":         "eu-central",
	"australia-southeast2":    "ap-southeast",
	"asia-south2":             "ap-south",
	"us-east7":                "us-east",
	"southamerica-west1":      "sa-west",
	"europe-west8":            "eu-west",
	// Oracle Mappings
	"us-ashburn-1":      "us-east",
	"eu-frankfurt-1":    "eu-central",
	"us-phoenix-1":      "us-west",
	"uk-london-1":       "eu-west",
	"sa-saopaulo-1":     "sa-east",
	"eu-amsterdam-1":    "eu-central",
	"ap-mumbai-1":       "ap-south",
	"ap-seoul-1":        "ap-northeast",
	"ap-tokyo-1":        "ap-northeast",
	"us-sanjose-1":      "us-west",
	"eu-zurich-1":       "eu-central",
	"ap-sydney-1":       "ap-southeast",
	"ap-hyderabad-1":    "ap-south",
	"ap-chuncheon-1":    "ap-northeast",
	"me-jeddah-1":       "me-south",
	"ap-osaka-1":        "ap-northeast",
	"ca-toronto-1":      "ca-east",
	"uk-cardiff-1":      "eu-west",
	"ap-singapore-1":    "ap-northeast",
	"ap-melbourne-1":    "ap-southeast",
	"sa-santiago-1":     "sa-west",
	"ca-montreal-1":     "ca-east",
	"me-dubai-1":        "ap-south",
	"eu-marseille-1":    "eu-west",
	"sa-vinhedo-1":      "sa-east",
	"me-abudhabi-1":     "me-south", // seems correct
	"il-jerusalem-1":    "me-south", // seems correct
	"eu-stockholm-1":    "eu-north",
	"eu-milan-1":        "eu-south", // seems correct
	"af-johannesburg-1": "af-south",
}

func normalizeRegion(region string) string {
	if v, ok := regionMap[region]; ok && v != "" {
		return v
	}

	re := regexp.MustCompile("\\-[0-9]")
	region = re.ReplaceAllString(region, "")

	return region
}

func (r *Range) MarshalJSON() ([]byte, error) {
	if r.Prefix.IP.To4() == nil {
		return nil, errors.New("does not support IPv6 addresses.")
	}

	last, _ := lastAddr(r.Prefix)

	return json.Marshal(&struct {
		Source     string
		Service    string
		Region     string
		RegionNorm string
		Prefix     string
		Start      uint32
		End        uint32
	}{
		Source:     r.Source,
		Service:    r.Service,
		Region:     r.Region,
		RegionNorm: normalizeRegion(r.Region),
		Prefix:     r.Prefix.String(),
		Start:      ip2uint32(r.Prefix.IP),
		End:        ip2uint32(last),
	})
}

type Ranges []*Range

func (r Ranges) Search(in string) *Range {
	ip := net.ParseIP(in)
	for _, r := range r {
		if r.Prefix.Contains(ip) {
			return r
		}
	}

	return nil
}

type awsFormat struct {
	SyncToken  string `json:"syncToken"`
	CreateDate string `json:"createDate"`
	Prefixes   []struct {
		IPPrefix string `json:"ip_prefix"`
		Region   string `json:"region"`
		Service  string `json:"service"`
		Border   string `json:"network_border_group"`
	} `json:"prefixes"`
}

type oracleFormat struct {
	Regions []struct {
		Region string `json:"region"`
		Cidrs  []struct {
			Cidr string   `json:"cidr"`
			Tags []string `json:"tags"`
		} `json:"cidrs"`
	} `json:"regions"`
}

type googleFormat struct {
	SyncToken  string `json:"syncToken"`
	CreateDate string `json:"creationTime"`
	Prefixes   []struct {
		IP4Prefix string `json:"ipv4Prefix"`
		Service   string `json:"service"`
		Scope     string `json:"scope"`
	} `json:"prefixes"`
}

type azureFormat struct {
	Cloud  string `json:"cloud"`
	Values []struct {
		Name       string `json:"name"`
		ID         string `json:"id"`
		Properties struct {
			Region          string `json:"region"`
			Platform        string `json:"platform"`
			Service         string `json:"systemService"`
			AddressPrefixes []string
		} `json:"properties"`
	}
}

func parseAws(r io.Reader) (Ranges, error) {
	var dat awsFormat
	ret := make(Ranges, 0)

	jd := json.NewDecoder(r)
	if err := jd.Decode(&dat); err != nil {
		return nil, err
	}

	for _, prefix := range dat.Prefixes {
		_, ipnet, err := net.ParseCIDR(prefix.IPPrefix)
		if err != nil {
			return nil, err
		}

		r := &Range{
			Source:  "AMAZON",
			Service: prefix.Service,
			Region:  prefix.Region,
			Prefix:  ipnet,
		}

		ret = append(ret, r)
	}

	return ret, nil
}

func parseOracle(r io.Reader) (Ranges, error) {
	var dat oracleFormat

	jd := json.NewDecoder(r)
	if err := jd.Decode(&dat); err != nil {
		return nil, err
	}

	ret := make(Ranges, 0)

	for _, ent := range dat.Regions {
		reg := ent.Region
		cidrs := ent.Cidrs
		for _, cidr := range cidrs {
			_, ipnet, err := net.ParseCIDR(cidr.Cidr)
			if err != nil {
				return nil, err
			}
			ret = append(ret, &Range{
				Source: "ORACLE",
				Region: reg,
				Prefix: ipnet,
			})
		}
	}

	return ret, nil

}

func parseAzure(r io.Reader) (Ranges, error) {
	var dat azureFormat

	jd := json.NewDecoder(r)
	if err := jd.Decode(&dat); err != nil {
		return nil, err
	}

	ret := make(Ranges, 0)

	for _, ent := range dat.Values {
		//src := ent.Properties.Platform
		svc := ent.Properties.Service
		reg := ent.Properties.Region

		if reg == "" {
			continue
		}

		for _, ip := range ent.Properties.AddressPrefixes {
			_, ipnet, err := net.ParseCIDR(ip)
			if err != nil {
				return nil, err
			}

			ret = append(ret, &Range{
				Source:  "AZURE",
				Service: svc,
				Region:  reg,
				Prefix:  ipnet,
			})
		}

	}

	return ret, nil
}

func parseGoogle(r io.Reader) (Ranges, error) {
	var dat googleFormat

	jd := json.NewDecoder(r)
	if err := jd.Decode(&dat); err != nil {
		return nil, err
	}

	ret := make(Ranges, 0)

	for _, prefix := range dat.Prefixes {
		if prefix.IP4Prefix == "" {
			continue
		}
		src := "GOOGLE"
		svc := prefix.Service
		reg := prefix.Scope

		_, ipnet, err := net.ParseCIDR(prefix.IP4Prefix)
		if err != nil {
			return nil, err
		}

		ret = append(ret, &Range{
			Source:  src,
			Service: svc,
			Region:  reg,
			Prefix:  ipnet,
		})
	}

	return ret, nil
}

type LineIterator struct {
	reader *bufio.Reader
}

func NewLineIterator(rd io.Reader) *LineIterator {
	return &LineIterator{
		reader: bufio.NewReader(rd),
	}
}

func (ln *LineIterator) Next() ([]byte, error) {
	var bytes []byte
	for {
		line, isPrefix, err := ln.reader.ReadLine()
		if err != nil {
			return nil, err
		}

		bytes = append(bytes, line...)
		if !isPrefix {
			break
		}
	}

	return bytes, nil
}

func main() {
	azure, err := os.Open("./azure-ranges.json")
	if err != nil {
		log.Fatal(err)
	}
	defer azure.Close()

	azureRanges, err := parseAzure(azure)
	if err != nil {
		log.Fatal(err)
	}

	aws, err := os.Open("./aws-ranges.json")
	if err != nil {
		log.Fatal(err)
	}
	defer aws.Close()

	awsRanges, err := parseAws(aws)
	if err != nil {
		log.Fatal(err)
	}

	goog, err := os.Open("./goog-ranges.json")
	if err != nil {
		log.Fatal(err)
	}
	defer goog.Close()

	googRanges, err := parseGoogle(goog)
	if err != nil {
		log.Fatal(err)
	}

	ora, err := os.Open("./oracle-ranges.json")
	if err != nil {
		log.Fatal(err)
	}
	defer ora.Close()

	oracleRanges, err := parseOracle(ora)
	if err != nil {
		log.Fatal(err)
	}

	all := make(Ranges, 0)
	all = append(all, azureRanges...)
	all = append(all, awsRanges...)
	all = append(all, googRanges...)
	all = append(all, oracleRanges...)

	for _, ent := range all {
		j, err := json.Marshal(ent)
		if err != nil {
			continue
		}

		fmt.Println(string(j))
	}
}
