// utg962e/utg962e.go
package utg962e

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/google/gousb"
)

const (
	vid = 0x6656
	pid = 0x0834
)

// SetFrequency отправляет команду на изменение частоты указанного канала
func SetFrequency(channel int, frequency, low, high float64) error {
	if channel != 1 && channel != 2 {
		return fmt.Errorf("channel must be 1 or 2")
	}
	if frequency < 0 || frequency > 60e6 {
		return fmt.Errorf("frequency must be in range 0-60e6 Hz")
	}

	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(vid, pid)
	if err != nil {
		return fmt.Errorf("cannot open device: %v", err)
	}
	if dev == nil {
		return fmt.Errorf("device not found")
	}
	defer dev.Close()
	dev.SetAutoDetach(true)

	cfg, err := dev.Config(1)
	if err != nil {
		return fmt.Errorf("Config(1) failed: %v", err)
	}
	defer cfg.Close()

	intf, err := cfg.Interface(0, 0)
	if err != nil {
		return fmt.Errorf("Interface(0,0) failed: %v", err)
	}
	defer intf.Close()

	epOut, err := intf.OutEndpoint(1)
	if err != nil {
		return fmt.Errorf("OutEndpoint(1) failed: %v", err)
	}

	// Формируем команду
	cmdStr := fmt.Sprintf(
		":CHAN%d:BASE:WAV SIN;:CHAN%d:BASE:FREQ %.1f;:CHAN%d:BASE:LOW %.1f;:CHAN%d:BASE:HIGH %.1f;:SYSTEM:LOCK OFF",
		channel, channel, frequency, channel, low, channel, high,
	)
	cmdBytes := []byte(cmdStr)

	// Префикс и суффикс
	prefix, _ := hex.DecodeString("0102FD006500000001000000")
	suffix, _ := hex.DecodeString("0D0A000000")

	fullCmd := append(prefix, cmdBytes...)
	fullCmd = append(fullCmd, suffix...)

	// Разбиваем на 64-байтные пакеты
	packets := chunkBytes(fullCmd, 64)

	// Отправка с проверкой готовности
	for i, pkg := range packets {
		for {
			if isDeviceReady(dev) {
				n, err := epOut.Write(pkg)
				if err != nil {
					log.Printf("Failed to send packet %d: %v, retrying...", i+1, err)
					time.Sleep(10 * time.Millisecond)
					continue
				}
				log.Printf("Sent packet %d: %d bytes", i+1, n)
				break
			} else {
				time.Sleep(10 * time.Millisecond)
			}
		}
	}

	return nil
}

// Проверка готовности устройства через Control transfer
func isDeviceReady(dev *gousb.Device) bool {
	buf := make([]byte, 1)
	_, err := dev.Control(0x80, 0x06, 0x0100, 0x0000, buf)
	return err == nil
}

// Разбивает срез байт на подмассивы фиксированного размера
func chunkBytes(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}
