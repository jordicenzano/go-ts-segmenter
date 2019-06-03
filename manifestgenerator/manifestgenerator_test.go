package manifestgenerator

import (
	"encoding/hex"
	"testing"
)

func parseHexString(h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic("bad test: " + h)
	}
	return b
}

func TestManifestGenerator(t *testing.T) {
	mg := New(false, ".", "chunk_", 4.0, LiveWindow, 3, 0)

	// Generate TS packet Pid = 1
	pckt := parseHexString("474011100042F0250001C10000FF01FF0001FC80144812010646466D70656709536572766963653031777C43CAFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")

	mg.AddData(pckt)

	procPckts := mg.getProcessedPackets()
	if procPckts != 1 {
		t.Errorf("Processed packet number is incorrect, got: %d, want: %d.", procPckts, 1)
	}
}
