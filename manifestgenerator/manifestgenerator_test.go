package manifestgenerator

import (
	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator/tspacket"
	"testing"
)

func TestManifestGenerator(t *testing.T) {
	mg := New(false, ".", "chunk_", 4.0, LiveWindow, 3, 0)

	// Generate TS packet filled with 0
	pckt := make([]byte, tspacket.TsDefaultPacketSize)
	pckt[0] = 0x47

	mg.AddData(pckt)

	procPckts := mg.getProcessedPackets()
	if procPckts != 1 {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, 1)
	}
}
