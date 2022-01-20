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

func (r *Range) MarshalJSON() ([]byte, error) {
	if r.Prefix.IP.To4() == nil {
		return nil, errors.New("does not support IPv6 addresses.")
	}

	last, _ := lastAddr(r.Prefix)

	return json.Marshal(&struct {
		Source  string
		Service string
		Region  string
		Prefix  string
		Start   uint32
		End     uint32
	}{
		Source:  r.Source,
		Service: r.Service,
		Region:  r.Region,
		Prefix:  r.Prefix.String(),
		Start:   ip2uint32(r.Prefix.IP),
		End:     ip2uint32(last),
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

func parseAzure(r io.Reader) (Ranges, error) {
	var dat azureFormat

	jd := json.NewDecoder(r)
	if err := jd.Decode(&dat); err != nil {
		return nil, err
	}

	ret := make(Ranges, 0)

	for _, ent := range dat.Values {
		src := ent.Properties.Platform
		svc := ent.Properties.Service
		reg := ent.Properties.Region

		for _, ip := range ent.Properties.AddressPrefixes {
			_, ipnet, err := net.ParseCIDR(ip)
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

	all := make(Ranges, 0)
	all = append(all, azureRanges...)
	all = append(all, awsRanges...)
	all = append(all, googRanges...)

	for _, ent := range all {
		j, err := json.Marshal(ent)
		if err != nil {
			continue
		}

		fmt.Println(string(j))
	}

	os.Exit(1)

	ln := NewLineIterator(os.Stdin)
	for {
		line, err := ln.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Fatal(err)
			}
		}

		if r := all.Search(string(line)); r != nil {
			ent := RangeEntry{InputIP: string(line), Range: r}
			j, err := json.Marshal(ent)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(j))
		}
	}
}
