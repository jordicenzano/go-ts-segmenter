package manifestgenerator

import (
	"bufio"
	"encoding/hex"
	"io"
	"os"
	"testing"
)

func parseHexString(h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic("bad test: " + h)
	}
	return b
}

func TestManifestGeneratorBasic1Pckt(t *testing.T) {
	mg := New(nil, false, ".", "chunk_", 4.0, 256, 257, LiveWindow, 3, 0)

	// Generate TS packet
	pckt := parseHexString("47410030075000007B0C7E00000001E0000080C00A310007EFD1110007D8610000000109F000000001674D4029965280A00B74A40404050000030001000003003C840000000168E90935200000000165888040006B6FFEF7D4B7CCB2D9A9BED82EA3DE8A78997D0DD494066F86757E1D7F4A3FA82C376EE9C0FE81F4F746A24E305C9A3E0DD5859DE0D287E8BEF70EA0CCF9008A25F52EF9A9CFA59B78AA5D34CB88001425FE7AB544EF7171FC56F27719F9C72D13FA7B0F5F3211A6")

	mg.AddData(pckt)

	xpectednumProcPackets := 1
	procPckts := mg.getProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}
}

func TestManifestGeneratorBasic2Pckt(t *testing.T) {
	mg := New(nil, false, ".", "chunk_", 4.0, 256, 257, LiveWindow, 3, 0)

	// Generate TS packet
	pckt := parseHexString(
		"47410030075000007B0C7E00000001E0000080C00A310007EFD1110007D8610000000109F000000001674D4029965280A00B74A40404050000030001000003003C840000000168E90935200000000165888040006B6FFEF7D4B7CCB2D9A9BED82EA3DE8A78997D0DD494066F86757E1D7F4A3FA82C376EE9C0FE81F4F746A24E305C9A3E0DD5859DE0D287E8BEF70EA0CCF9008A25F52EF9A9CFA59B78AA5D34CB88001425FE7AB544EF7171FC56F27719F9C72D13FA7B0F5F3211A6" +
			"47410030075000007B0C7E00000001E0000080C00A310007EFD1110007D8610000000109F000000001674D4029965280A00B74A40404050000030001000003003C840000000168E90935200000000165888040006B6FFEF7D4B7CCB2D9A9BED82EA3DE8A78997D0DD494066F86757E1D7F4A3FA82C376EE9C0FE81F4F746A24E305C9A3E0DD5859DE0D287E8BEF70EA0CCF9008A25F52EF9A9CFA59B78AA5D34CB88001425FE7AB544EF7171FC56F27719F9C72D13FA7B0F5F3211A6")

	mg.AddData(pckt)

	xpectednumProcPackets := 2
	procPckts := mg.getProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}
}

func TestManifestGeneratorBasicVideoBigPackets(t *testing.T) {
	f, err := os.Open("../fixture/test.ts")
	if err != nil {
		panic("Error opening test file")
	}

	mediaSourceReader := bufio.NewReader(f)
	buf := make([]byte, 0, 4*1024) //4KB Buffers

	mg := New(nil, false, ".", "chunk_", 4.0, 256, 257, LiveWindow, 3, 0)

	for {
		n, err := mediaSourceReader.Read(buf[:cap(buf)])
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
		} else {
			mg.AddData(buf)
		}
		// process buf
		if err != nil && err != io.EOF {
			panic("Error reading test file")
		}
	}
	xpectednumProcPackets := 14486
	procPckts := mg.getProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}
}

func TestManifestGeneratorBasicVideoSmallPackets(t *testing.T) {
	f, err := os.Open("../fixture/test.ts")
	if err != nil {
		panic("Error opening test file")
	}

	mediaSourceReader := bufio.NewReader(f)
	buf := make([]byte, 0, 100) //100 bytes

	mg := New(nil, false, ".", "chunk_", 4.0, 256, 257, LiveWindow, 3, 0)

	for {
		n, err := mediaSourceReader.Read(buf[:cap(buf)])
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
		} else {
			mg.AddData(buf)
		}
		// process buf
		if err != nil && err != io.EOF {
			panic("Error reading test file")
		}
	}
	xpectednumProcPackets := 14486
	procPckts := mg.getProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}
}
