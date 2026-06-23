package main

import (
	"fmt"
	"log"
	"os"

	"github.com/anviod/fins/udp"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: udp-client <plc_ip> <plc_port>")
		fmt.Println("Example: udp-client 127.0.0.1 9600")
		os.Exit(1)
	}

	plcIP := os.Args[1]
	plcPort := os.Args[2]

	fmt.Printf("=== Omron FINS UDP Client Test ===\n")
	fmt.Printf("PLC IP: %s, Port: %s\n\n", plcIP, plcPort)

	clientAddr := udp.NewAddress("", 0, 0, 2, 255)
	plcAddr := udp.NewAddress(plcIP, 9600, 0, 1, 0)

	s, err := udp.NewPLCSimulator(plcAddr)
	if err != nil {
		log.Fatalf("Failed to start PLC simulator: %v", err)
	}
	defer s.Close()
	fmt.Println("PLC Simulator started")

	c, err := udp.NewClient(clientAddr, plcAddr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()
	fmt.Println("Client connected successfully")

	testCases := []struct {
		name string
		fn   func() error
	}{
		{"Test Write/Read Words", testWriteReadWords(c)},
		{"Test Write/Read Bits", testWriteReadBits(c)},
		{"Test Write/Read Bytes", testWriteReadBytes(c)},
		{"Test Write/Read String", testWriteReadString(c)},
		{"Test Read Clock", testReadClock(c)},
	}

	fmt.Println("\n--- Running Tests ---")
	passCount := 0
	failCount := 0

	for _, tc := range testCases {
		fmt.Printf("\n[%s]\n", tc.name)
		if err := tc.fn(); err != nil {
			fmt.Printf("FAIL: %v\n", err)
			failCount++
		} else {
			fmt.Printf("PASS\n")
			passCount++
		}
	}

	fmt.Println("\n--- Test Summary ---")
	fmt.Printf("Total: %d, Passed: %d, Failed: %d\n", len(testCases), passCount, failCount)

	if failCount > 0 {
		os.Exit(1)
	}
	fmt.Println("\nAll tests passed!")
}

func testWriteReadWords(c *udp.Client) func() error {
	return func() error {
		toWrite := []uint16{1234, 5678, 9012, 3456, 7890}
		if err := c.WriteWords(udp.MemoryAreaDMWord, 100, toWrite); err != nil {
			return fmt.Errorf("WriteWords failed: %v", err)
		}

		vals, err := c.ReadWords(udp.MemoryAreaDMWord, 100, 5)
		if err != nil {
			return fmt.Errorf("ReadWords failed: %v", err)
		}

		for i, v := range vals {
			if v != toWrite[i] {
				return fmt.Errorf("Value mismatch at index %d: expected %d, got %d", i, toWrite[i], v)
			}
		}
		fmt.Printf("  Wrote: %v\n", toWrite)
		fmt.Printf("  Read:  %v\n", vals)
		return nil
	}
}

func testWriteReadBits(c *udp.Client) func() error {
	return func() error {
		toWrite := []bool{true, false, true, true, false}
		if err := c.WriteBits(udp.MemoryAreaDMBit, 200, 0, toWrite); err != nil {
			return fmt.Errorf("WriteBits failed: %v", err)
		}

		vals, err := c.ReadBits(udp.MemoryAreaDMBit, 200, 0, 5)
		if err != nil {
			return fmt.Errorf("ReadBits failed: %v", err)
		}

		for i, v := range vals {
			if v != toWrite[i] {
				return fmt.Errorf("Value mismatch at index %d: expected %v, got %v", i, toWrite[i], v)
			}
		}
		fmt.Printf("  Wrote: %v\n", toWrite)
		fmt.Printf("  Read:  %v\n", vals)
		return nil
	}
}

func testWriteReadBytes(c *udp.Client) func() error {
	return func() error {
		toWrite := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
		if err := c.WriteBytes(udp.MemoryAreaDMWord, 300, toWrite); err != nil {
			return fmt.Errorf("WriteBytes failed: %v", err)
		}

		vals, err := c.ReadBytes(udp.MemoryAreaDMWord, 300, uint16(len(toWrite)/2))
		if err != nil {
			return fmt.Errorf("ReadBytes failed: %v", err)
		}

		for i, v := range vals {
			if v != toWrite[i] {
				return fmt.Errorf("Value mismatch at index %d: expected 0x%02x, got 0x%02x", i, toWrite[i], v)
			}
		}
		fmt.Printf("  Wrote: %v\n", toWrite)
		fmt.Printf("  Read:  %v\n", vals)
		return nil
	}
}

func testWriteReadString(c *udp.Client) func() error {
	return func() error {
		testStr := "Hello Omron FINS UDP"
		if err := c.WriteString(udp.MemoryAreaDMWord, 400, testStr); err != nil {
			return fmt.Errorf("WriteString failed: %v", err)
		}

		v, err := c.ReadString(udp.MemoryAreaDMWord, 400, uint16(len(testStr)))
		if err != nil {
			return fmt.Errorf("ReadString failed: %v", err)
		}

		if v != testStr {
			return fmt.Errorf("String mismatch: expected %q, got %q", testStr, v)
		}
		fmt.Printf("  Wrote: %q\n", testStr)
		fmt.Printf("  Read:  %q\n", v)
		return nil
	}
}

func testReadClock(c *udp.Client) func() error {
	return func() error {
		t, err := c.ReadClock()
		if err != nil {
			return fmt.Errorf("ReadClock failed: %v", err)
		}
		fmt.Printf("  Clock: %s\n", t.Format("2006-01-02 15:04:05"))
		return nil
	}
}