package tspacket

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

func TestTSPacketPID(t *testing.T) {
	tsPckt := New(TsDefaultPacketSize)

	// Generate TS packet Pid = 1
	buf := parseHexString("474011100042F0250001C10000FF01FF0001FC80144812010646466D70656709536572766963653031777C43CAFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	tsPckt.AddData(buf)

	xpectedPid := 17
	if pid := tsPckt.GetPID(); pid != xpectedPid {
		t.Errorf("Pid is not correct, got = %d, want %d", pid, xpectedPid)
	}

	if pcrS := tsPckt.GetPCRS(); pcrS != -1 {
		t.Errorf("IDR is not correct, got = %f, want %f", pcrS, 0.0)
	}

	if isIDR := tsPckt.GetIDR(); isIDR != false {
		t.Errorf("IDR is not correct, got = %t, want %t", isIDR, false)
	}
}
