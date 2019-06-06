package manifestgenerator

import (
	"bufio"
	"encoding/hex"
	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator/hls"
	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator/mediachunk"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func parseHexString(h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic("bad test: " + h)
	}
	return b
}

// Clear directory results
func clearResultsDir(pathResults string) {
	os.RemoveAll(pathResults)

	os.MkdirAll(pathResults, 0744)
}

func TestManifestGeneratorBasic1Pckt(t *testing.T) {
	pathResults := "../results/Basic1Pckt"
	clearResultsDir(pathResults)

	mg := New(nil, mediachunk.OutputModeNone, pathResults, "chunk_", "chunklist.m3u8", 4.0, ChunkNoIni, false, 256, 257, hls.LiveWindow, 3, 0)

	// Generate TS packet
	pckt := parseHexString("47410030075000007B0C7E00000001E0000080C00A310007EFD1110007D8610000000109F000000001674D4029965280A00B74A40404050000030001000003003C840000000168E90935200000000165888040006B6FFEF7D4B7CCB2D9A9BED82EA3DE8A78997D0DD494066F86757E1D7F4A3FA82C376EE9C0FE81F4F746A24E305C9A3E0DD5859DE0D287E8BEF70EA0CCF9008A25F52EF9A9CFA59B78AA5D34CB88001425FE7AB544EF7171FC56F27719F9C72D13FA7B0F5F3211A6")

	mg.AddData(pckt)

	//mg.Close()

	xpectednumProcPackets := uint64(1)
	procPckts := mg.getNumProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}
}

func TestManifestGeneratorBasic2Pckt(t *testing.T) {
	pathResults := "../results/Basic2Pckt"
	clearResultsDir(pathResults)

	mg := New(nil, mediachunk.OutputModeNone, pathResults, "chunk_", "chunklist.m3u8", 4.0, ChunkNoIni, false, 256, 257, hls.LiveWindow, 3, 0)

	// Generate TS packet
	pckt := parseHexString(
		"47410030075000007B0C7E00000001E0000080C00A310007EFD1110007D8610000000109F000000001674D4029965280A00B74A40404050000030001000003003C840000000168E90935200000000165888040006B6FFEF7D4B7CCB2D9A9BED82EA3DE8A78997D0DD494066F86757E1D7F4A3FA82C376EE9C0FE81F4F746A24E305C9A3E0DD5859DE0D287E8BEF70EA0CCF9008A25F52EF9A9CFA59B78AA5D34CB88001425FE7AB544EF7171FC56F27719F9C72D13FA7B0F5F3211A6" +
			"47410030075000007B0C7E00000001E0000080C00A310007EFD1110007D8610000000109F000000001674D4029965280A00B74A40404050000030001000003003C840000000168E90935200000000165888040006B6FFEF7D4B7CCB2D9A9BED82EA3DE8A78997D0DD494066F86757E1D7F4A3FA82C376EE9C0FE81F4F746A24E305C9A3E0DD5859DE0D287E8BEF70EA0CCF9008A25F52EF9A9CFA59B78AA5D34CB88001425FE7AB544EF7171FC56F27719F9C72D13FA7B0F5F3211A6")

	mg.AddData(pckt)

	mg.Close()

	xpectednumProcPackets := uint64(2)
	procPckts := mg.getNumProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}
}

func TestManifestGeneratorBasicVideoBigPacketsNoIniData(t *testing.T) {
	pathResults := "../results/VideoBigPackets"
	clearResultsDir(pathResults)

	f, err := os.Open("../fixture/testSmall.ts")
	if err != nil {
		panic("Error opening test file")
	}

	mediaSourceReader := bufio.NewReader(f)
	buf := make([]byte, 0, 4*1024) //4KB Buffers

	mg := New(nil, mediachunk.OutputModeNone, pathResults, "chunk_", "chunklist.m3u8", 4.0, ChunkNoIni, false, 256, 257, hls.LiveWindow, 3, 0)

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
	mg.Close()

	xpectednumProcPackets := uint64(1835)
	procPckts := mg.getNumProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}
}

func TestManifestGeneratorBasicVideoSmallPacketsNoIniData(t *testing.T) {
	pathResults := "../results/VideoSmallPackets"
	clearResultsDir(pathResults)

	f, err := os.Open("../fixture/testSmall.ts")
	if err != nil {
		panic("Error opening test file")
	}

	mediaSourceReader := bufio.NewReader(f)
	buf := make([]byte, 0, 100) //100 bytes

	mg := New(nil, mediachunk.OutputModeNone, pathResults, "chunk_", "chunklist.m3u8", 4.0, ChunkNoIni, false, 256, 257, hls.LiveWindow, 3, 0)

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
	mg.Close()

	xpectednumProcPackets := uint64(1835)
	procPckts := mg.getNumProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}
}

func TestManifestGeneratorInitialResyncVideoBigPacketsNoIniData(t *testing.T) {
	pathResults := "../results/VideoResyncBigPackets"
	clearResultsDir(pathResults)

	f, err := os.Open("../fixture/testSmall.ts")
	if err != nil {
		panic("Error opening test file")
	}

	mediaSourceReader := bufio.NewReader(f)
	buf := make([]byte, 0, 4*1024) //4KB Buffers

	mg := New(nil, mediachunk.OutputModeNone, pathResults, "chunk_", "chunklist.m3u8", 4.0, ChunkNoIni, false, 256, 257, hls.LiveWindow, 3, 0)

	// Start out of sync
	n, err := mediaSourceReader.Read(buf[:cap(buf)])
	if err != nil {
		panic("Error reading test file")
	}

	for {
		n, err = mediaSourceReader.Read(buf[:cap(buf)])
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
	mg.Close()

	xpectednumProcPackets := uint64(1813)
	procPckts := mg.getNumProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}
}

func TestManifestGeneratorBasicVideoBigPacketsAutoPIDsInitSegment(t *testing.T) {
	pathResults := "../results/VideoBigPacketsAutoPIDsInitSegment"
	clearResultsDir(pathResults)

	f, err := os.Open("../fixture/testSmall.ts")
	if err != nil {
		panic("Error opening test file")
	}

	mediaSourceReader := bufio.NewReader(f)
	buf := make([]byte, 0, 4*1024) //4KB Buffers

	mg := New(nil, mediachunk.OutputModeFile, pathResults, "chunk_", "chunklist.m3u8", 4.0, ChunkInit, true, -1, -1, hls.LiveWindow, 3, 0)

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
	mg.Close()

	xpectednumProcPackets := uint64(1835)
	procPckts := mg.getNumProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}

	// Check chunks
	type fileData struct {
		name string
		size int64
	}

	filesData := []fileData{
		{
			name: path.Join(pathResults, "init00000.ts"),
			size: 376,
		},
		{
			name: path.Join(pathResults, "chunk_00000.ts"),
			size: 97760,
		},
		{
			name: path.Join(pathResults, "chunk_00001.ts"),
			size: 103212,
		},
		{
			name: path.Join(pathResults, "chunk_00002.ts"),
			size: 108100,
		},
		{
			name: path.Join(pathResults, "chunk_00003.ts"),
			size: 16920,
		},
	}

	for _, filesToCheck := range filesData {
		fi, err := os.Stat(filesToCheck.name)
		if err != nil || fi.Size() != filesToCheck.size {
			t.Errorf("Error checking file %s, got %d bytes, expected %d bytes. Err: %v", filesToCheck.name, fi.Size(), filesToCheck.size, err)
		}
	}
}

func TestManifestGeneratorBasicVideoBigPacketsAutoPIDsInitStartSegment(t *testing.T) {
	pathResults := "../results/VideoBigPacketsAutoPIDsInitStartSegment"
	chuklistFile := "chunklist.m3u8"
	clearResultsDir(pathResults)

	f, err := os.Open("../fixture/testSmall.ts")
	if err != nil {
		panic("Error opening test file")
	}

	mediaSourceReader := bufio.NewReader(f)
	buf := make([]byte, 0, 4*1024) //4KB Buffers

	mg := New(nil, mediachunk.OutputModeFile, pathResults, "chunk_", chuklistFile, 4.0, ChunkInitStart, true, -1, -1, hls.LiveWindow, 3, 0)

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
	mg.Close()

	xpectednumProcPackets := uint64(1835)
	procPckts := mg.getNumProcessedPackets()
	if procPckts != xpectednumProcPackets {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, xpectednumProcPackets)
	}

	// Check chunks
	type fileData struct {
		name string
		size int64
	}

	filesData := []fileData{
		{
			name: path.Join(pathResults, "chunk_00000.ts"),
			size: 98136,
		},
		{
			name: path.Join(pathResults, "chunk_00001.ts"),
			size: 103588,
		},
		{
			name: path.Join(pathResults, "chunk_00002.ts"),
			size: 108476,
		},
		{
			name: path.Join(pathResults, "chunk_00003.ts"),
			size: 17296,
		},
	}

	for _, filesToCheck := range filesData {
		fi, err := os.Stat(filesToCheck.name)
		if err != nil || fi.Size() != filesToCheck.size {
			t.Errorf("Error checking file %s, got %d bytes, expected %d bytes. Err: %v", filesToCheck.name, fi.Size(), filesToCheck.size, err)
		}
	}

	// Check HLS chunklist
	fileChunklistHLS, err := os.Open(path.Join(pathResults, chuklistFile))
	if err != nil {
		t.Errorf("Error opening HLS chunklist!, Err: %v", err)
	}
	defer fileChunklistHLS.Close()

	manifestByte, err := ioutil.ReadAll(fileChunklistHLS)
	if err != nil {
		t.Errorf("Error reading HLS chunklist data!, Err: %v", err)
	}

	manifestStr := string(manifestByte)
	xpectedmanifestStr := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:1
#EXT-X-DISCONTINUITY-SEQUENCE:0
#EXT-X-TARGETDURATION:4.00000000
#EXT-X-INDEPENDENT-SEGMENTS
#EXTINF:3.80000000,
chunk_00001.ts
#EXTINF:3.80000000,
chunk_00002.ts
#EXTINF:0.50000000,
chunk_00003.ts
`
	if manifestStr != xpectedmanifestStr {
		t.Errorf("Manifest data is different, got %s , expected %s", manifestStr, xpectedmanifestStr)
	}
}
