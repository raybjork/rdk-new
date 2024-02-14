// Package rtkutils implements necessary functions to set and return
// NTRIP information here.
package rtkutils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/de-bkg/gognss/pkg/ntrip"

	"go.viam.com/rdk/logging"
)

const (
	// id numbers of the different fields returned in the standard
	// Stream response from the ntrip client, numbered 1-18.
	// Information on each field is explained int the comments
	// of the Stream struct.
	mp            = 1
	id            = 2
	format        = 3
	formatDetails = 4
	carrierField  = 5
	navsystem     = 6
	network       = 7
	country       = 8
	latitude      = 9
	longitude     = 10
	nmeaBit       = 11
	solution      = 12
	generator     = 13
	compression   = 14
	auth          = 15
	feeBit        = 16
	bitRateField  = 17
	misc          = 18
	floatbitsize  = 32
	streamSize    = 200
)

// NtripInfo contains the information necessary to connect to a mountpoint.
type NtripInfo struct {
	URL                string
	username           string
	password           string
	MountPoint         string
	Client             *ntrip.Client
	Stream             io.ReadCloser
	MaxConnectAttempts int
}

// NtripConfig is used for converting attributes for a correction source.
type NtripConfig struct {
	NtripURL             string `json:"ntrip_url"`
	NtripConnectAttempts int    `json:"ntrip_connect_attempts,omitempty"`
	NtripMountpoint      string `json:"ntrip_mountpoint,omitempty"`
	NtripUser            string `json:"ntrip_username,omitempty"`
	NtripPass            string `json:"ntrip_password,omitempty"`
}

// Sourcetable struct contains the stream.
type Sourcetable struct {
	Streams []Stream
}

// Stream contrains a stream record in sourcetable.
// https://software.rtcm-ntrip.org/wiki/STR
type Stream struct {
	MP             string   // Datastream mountpoint
	Identifier     string   // Source identifier (most time nearest city)
	Format         string   // Data format of generic type (https://software.rtcm-ntrip.org/wiki/STR#DataFormats)
	FormatDetails  string   // Specifics of data format (https://software.rtcm-ntrip.org/wiki/STR#DataFormats)
	Carrier        int      // Phase information about GNSS correction (https://software.rtcm-ntrip.org/wiki/STR#Carrier)
	NavSystem      []string // Multiple navigation system (https://software.rtcm-ntrip.org/wiki/STR#NavigationSystem)
	Network        string   // Network record in sourcetable (https://software.rtcm-ntrip.org/wiki/NET)
	Country        string   // ISO 3166 country code (https://en.wikipedia.org/wiki/ISO_3166-1)
	Latitude       float32  // Position, Latitude in degree
	Longitude      float32  // Position, Longitude in degree
	Nmea           bool     // Caster requires NMEA input (1) or not (0)
	Solution       int      // Generated by single base (0) or network (1)
	Generator      string   // Generating soft- or hardware
	Compression    string   // Compression algorithm
	Authentication string   // Access protection for data streams None (N), Basic (B) or Digest (D)
	Fee            bool     // User fee for data access: yes (Y) or no (N)
	BitRate        int      // Datarate in bits per second
	Misc           string   // Miscellaneous information
}

// NewNtripInfo function validates and sets NtripConfig arributes and returns NtripInfo.
func NewNtripInfo(cfg *NtripConfig, logger logging.Logger) (*NtripInfo, error) {
	n := &NtripInfo{}

	// Init NtripInfo from attributes
	n.URL = cfg.NtripURL
	if n.URL == "" {
		return nil, fmt.Errorf("NTRIP expected non-empty string for %q", cfg.NtripURL)
	}
	n.username = cfg.NtripUser
	if n.username == "" {
		logger.Info("ntrip_username set to empty")
	}
	n.password = cfg.NtripPass
	if n.password == "" {
		logger.Info("ntrip_password set to empty")
	}
	n.MountPoint = cfg.NtripMountpoint
	if n.MountPoint == "" {
		logger.Info("ntrip_mountpoint set to empty")
	}
	n.MaxConnectAttempts = cfg.NtripConnectAttempts
	if n.MaxConnectAttempts == 0 {
		logger.Info("ntrip_connect_attempts using default 10")
		n.MaxConnectAttempts = 10
	}

	logger.Debug("Returning n")
	return n, nil
}

// ParseSourcetable gets the sourcetable and parses it.
func (n *NtripInfo) ParseSourcetable(logger logging.Logger) (*Sourcetable, error) {
	reader, err := n.Client.GetSourcetable()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			logger.Errorf("Error closing reader:", err)
		}
	}()

	st := &Sourcetable{}
	st.Streams = make([]Stream, 0, streamSize)
	scanner := bufio.NewScanner(reader)

Loop:
	for scanner.Scan() {
		ln := scanner.Text()

		// Check if the line is a comment.
		if strings.HasPrefix(ln, "#") || strings.HasPrefix(ln, "*") {
			continue
		}
		fields := strings.Split(ln, ";")
		switch fields[0] {
		case "CAS":
			continue
		case "NET":
			continue
		case "STR":
			str, err := parseStream(ln)
			if err != nil {
				return nil, fmt.Errorf("error while parsing stream: %w", err)
			}
			st.Streams = append(st.Streams, str)
		case "ENDSOURCETABLE":
			break Loop
		default:
			return nil, fmt.Errorf("%s: illegal sourcetable line: '%s'", n.URL, ln)
		}
	}

	return st, nil
}

// ParseStream parses a line from the sourcetable.
func parseStream(line string) (Stream, error) {
	fields := strings.Split(line, ";")

	// Standard stream contains 19 fields.
	// They are enumerated by their constants at the top of the file
	if len(fields) < 19 {
		return Stream{}, fmt.Errorf("missing fields at stream line: %s", line)
	}

	carrier, err := strconv.Atoi(fields[carrierField])
	if err != nil {
		return Stream{}, fmt.Errorf("cannot parse the streams carrier in line: %s", line)
	}

	satSystems := strings.Split(fields[navsystem], "+")

	lat, err := strconv.ParseFloat(fields[latitude], floatbitsize)
	if err != nil {
		return Stream{}, fmt.Errorf("cannot parse the streams latitude in line: %s", line)
	}
	lon, err := strconv.ParseFloat(fields[longitude], floatbitsize)
	if err != nil {
		return Stream{}, fmt.Errorf("cannot parse the streams longitude in line: %s", line)
	}

	nmea, err := strconv.ParseBool(fields[nmeaBit])
	if err != nil {
		return Stream{}, fmt.Errorf("cannot parse the streams nmea in line: %s", line)
	}

	sol, err := strconv.Atoi(fields[solution])
	if err != nil {
		return Stream{}, fmt.Errorf("cannot parse the streams solution in line: %s", line)
	}

	fee := false
	if fields[feeBit] == "Y" {
		fee = true
	}

	bitrate, err := strconv.Atoi(fields[bitRateField])
	if err != nil {
		bitrate = 0
	}

	return Stream{
		MP: fields[mp], Identifier: fields[id], Format: fields[format], FormatDetails: fields[formatDetails],
		Carrier: carrier, NavSystem: satSystems, Network: fields[network], Country: fields[country],
		Latitude: float32(lat), Longitude: float32(lon), Nmea: nmea, Solution: sol, Generator: fields[generator],
		Compression: fields[compression], Authentication: fields[auth], Fee: fee, BitRate: bitrate, Misc: fields[misc],
	}, nil
}

// Connect attempts to initialize a new ntrip client. If we're unable to connect after multiple
// attempts, we return the last error.
func (n *NtripInfo) Connect(ctx context.Context, logger logging.Logger) error {
	var c *ntrip.Client
	var err error

	logger.Debug("Connecting to NTRIP caster")
	for attempts := 0; attempts < n.MaxConnectAttempts; attempts++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		c, err = ntrip.NewClient(n.URL, ntrip.Options{Username: n.username, Password: n.password})
		if err == nil { // Success!
			logger.Info("Connected to NTRIP caster")
			n.Client = c
			return nil
		}
	}

	logger.Errorf("Can't connect to NTRIP caster: %s", err)
	return err
}

// HasStream checks if the sourcetable contains the given mountpoint in it's stream.
func (st *Sourcetable) HasStream(mountpoint string) (Stream, bool) {
	for _, str := range st.Streams {
		if str.MP == mountpoint {
			return str, true
		}
	}

	return Stream{}, false
}
