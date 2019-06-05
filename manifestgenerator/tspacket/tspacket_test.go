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

func TestTSPacketPIDNoIDR(t *testing.T) {
	tsPckt := New(TsDefaultPacketSize)

	// Generate TS packet
	buf := parseHexString("474011100042F0250001C10000FF01FF0001FC80144812010646466D70656709536572766963653031777C43CAFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	tsPckt.AddData(buf)
	tsPckt.Parse(-1)

	videoPid := 17

	xpectedPid := videoPid
	if pid := tsPckt.GetPID(); pid != xpectedPid {
		t.Errorf("Pid is not correct, got = %d, want %d", pid, xpectedPid)
	}

	xpectedPCRS := -1.0
	if pcrS := tsPckt.GetPCRS(); pcrS != xpectedPCRS {
		t.Errorf("PCR is not correct, got = %f, want %f", pcrS, xpectedPCRS)
	}

	xpectedisRandomAccess := false
	if isRandomAccess := tsPckt.IsRandomAccess(videoPid); isRandomAccess != xpectedisRandomAccess {
		t.Errorf("RandomAccess is not correct, got = %t, want %t", isRandomAccess, xpectedisRandomAccess)
	}
}

func TestTSPacketPIDIDRPCR(t *testing.T) {
	tsPckt := New(TsDefaultPacketSize)

	// Generate TS packet
	buf := parseHexString("47410030075000007B0C7E00000001E0000080C00A310007EFD1110007D8610000000109F000000001674D4029965280A00B74A40404050000030001000003003C840000000168E90935200000000165888040006B6FFEF7D4B7CCB2D9A9BED82EA3DE8A78997D0DD494066F86757E1D7F4A3FA82C376EE9C0FE81F4F746A24E305C9A3E0DD5859DE0D287E8BEF70EA0CCF9008A25F52EF9A9CFA59B78AA5D34CB88001425FE7AB544EF7171FC56F27719F9C72D13FA7B0F5F3211A6")
	tsPckt.AddData(buf)
	tsPckt.Parse(-1)

	videoPid := 256

	xpectedPid := videoPid
	if pid := tsPckt.GetPID(); pid != xpectedPid {
		t.Errorf("Pid is not correct, got = %d, want %d", pid, xpectedPid)
	}

	xpectedPCRS := 0.7
	if pcrS := tsPckt.GetPCRS(); pcrS != xpectedPCRS {
		t.Errorf("IDR is not correct, got = %f, want %f", pcrS, xpectedPCRS)
	}

	xpectedisRandomAccess := true
	if isRandomAccess := tsPckt.IsRandomAccess(videoPid); isRandomAccess != xpectedisRandomAccess {
		t.Errorf("RandomAccess is not correct, got = %t, want %t", isRandomAccess, xpectedisRandomAccess)
	}
}
