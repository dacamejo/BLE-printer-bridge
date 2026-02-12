package ble

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

var Adapter = bluetooth.DefaultAdapter

const scanStopGrace = 2 * time.Second

var (
	scanMu         sync.Mutex
	scanInProgress bool
	ErrScanBusy    = errors.New("bluetooth scan already in progress")
	ErrScanTimeout = errors.New("bluetooth scan timed out while stopping")
)

type ScanHit struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	RSSI    int16  `json:"rssi"`
}

type CharacteristicInfo struct {
	UUID                 string `json:"uuid"`
	Write                bool   `json:"write"`
	WriteWithoutResponse bool   `json:"write_without_response"`
	Notify               bool   `json:"notify"`
	Read                 bool   `json:"read"`
}

type ServiceInfo struct {
	UUID            string               `json:"uuid"`
	Characteristics []CharacteristicInfo `json:"characteristics"`
}

type DescribeResult struct {
	Services []ServiceInfo `json:"services"`
}

type Client struct {
	mu        sync.Mutex
	dev       bluetooth.Device
	connected bool
}

func Enable() error { return Adapter.Enable() }

func Scan(seconds int, nameContains string) ([]ScanHit, error) {
	if seconds <= 0 {
		seconds = 8
	}

	scanMu.Lock()
	if scanInProgress {
		scanMu.Unlock()
		return nil, ErrScanBusy
	}
	scanInProgress = true
	scanMu.Unlock()
	defer func() {
		scanMu.Lock()
		scanInProgress = false
		scanMu.Unlock()
	}()

	var (
		hits   []ScanHit
		hitsMu sync.Mutex
	)

	scanDone := make(chan error, 1)
	go func() {
		err := Adapter.Scan(func(a *bluetooth.Adapter, r bluetooth.ScanResult) {
			name := r.LocalName()
			if nameContains != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(nameContains)) {
				return
			}
			hitsMu.Lock()
			hits = append(hits, ScanHit{Address: r.Address.String(), Name: name, RSSI: r.RSSI})
			hitsMu.Unlock()
		})
		scanDone <- err
	}()

	time.Sleep(time.Duration(seconds) * time.Second)
	_ = Adapter.StopScan()

	var err error
	select {
	case err = <-scanDone:
	case <-time.After(scanStopGrace):
		return nil, fmt.Errorf("%w after %s scan window", ErrScanTimeout, time.Duration(seconds)*time.Second)
	}
	if err != nil {
		return nil, err
	}

	hitsMu.Lock()
	defer hitsMu.Unlock()
	out := make([]ScanHit, len(hits))
	copy(out, hits)
	return out, nil
}

func (c *Client) Connect(address string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cleanAddress, err := normalizeAddress(address)
	if err != nil {
		return err
	}

	if c.connected {
		_ = c.dev.Disconnect()
		c.connected = false
	}
	a := bluetooth.Address{}
	a.Set(cleanAddress)
	dev, err := Adapter.Connect(a, bluetooth.ConnectionParams{})
	if err != nil {
		return err
	}
	if _, err := dev.DiscoverServices(nil); err != nil {
		_ = dev.Disconnect()
		return fmt.Errorf("connected but could not verify link: %w", err)
	}
	c.dev = dev
	c.connected = true
	return nil
}

func normalizeAddress(address string) (string, error) {
	cleaned := strings.ToUpper(strings.TrimSpace(address))
	cleaned = strings.ReplaceAll(cleaned, "-", ":")
	parts := strings.Split(cleaned, ":")
	if len(parts) != 6 {
		return "", fmt.Errorf("invalid device address format: %q", address)
	}
	for _, part := range parts {
		if len(part) != 2 {
			return "", fmt.Errorf("invalid device address format: %q", address)
		}
		if _, err := hex.DecodeString(part); err != nil {
			return "", fmt.Errorf("invalid device address format: %q", address)
		}
	}
	return cleaned, nil
}

func NormalizeAddress(address string) (string, error) {
	return normalizeAddress(address)
}

func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return false
	}
	if _, err := c.dev.DiscoverServices(nil); err != nil {
		_ = c.dev.Disconnect()
		c.connected = false
	}
	return c.connected
}

func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}
	if err := c.dev.Disconnect(); err != nil {
		return err
	}
	c.connected = false
	return nil
}

func (c *Client) Describe() (*DescribeResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil, errors.New("not connected")
	}
	services, err := c.dev.DiscoverServices(nil)
	if err != nil {
		return nil, err
	}
	out := &DescribeResult{}
	for _, s := range services {
		si := ServiceInfo{UUID: s.UUID().String()}
		chars, err := s.DiscoverCharacteristics(nil)
		if err != nil {
			continue
		}
		for _, ch := range chars {
			props := uint32(ch.Properties())
			si.Characteristics = append(si.Characteristics, CharacteristicInfo{
				UUID:                 ch.UUID().String(),
				Read:                 (props & 0x02) != 0,
				WriteWithoutResponse: (props & 0x04) != 0,
				Write:                (props & 0x08) != 0,
				Notify:               (props & 0x10) != 0,
			})
		}
		out.Services = append(out.Services, si)
	}
	return out, nil
}

func (c *Client) Print(serviceUUID, charUUID string, data []byte, chunkSize int, withResponse bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return errors.New("not connected")
	}
	if chunkSize <= 0 {
		chunkSize = 180
	}

	su, err := bluetooth.ParseUUID(serviceUUID)
	if err != nil {
		return err
	}
	cu, err := bluetooth.ParseUUID(charUUID)
	if err != nil {
		return err
	}

	services, err := c.dev.DiscoverServices([]bluetooth.UUID{su})
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return errors.New("service not found")
	}

	chars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{cu})
	if err != nil {
		return err
	}
	if len(chars) == 0 {
		return errors.New("characteristic not found")
	}
	ch := chars[0]

	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		part := data[i:end]
		if withResponse {
			_, err = ch.Write(part)
		} else {
			_, err = ch.WriteWithoutResponse(part)
		}
		if err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}
