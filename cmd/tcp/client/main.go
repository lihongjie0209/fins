package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/anviod/fins"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: tcp-client <plc_ip> <plc_port>")
		fmt.Println("Example: tcp-client 127.0.0.1 9600")
		os.Exit(1)
	}

	plcIP := os.Args[1]
	plcPortStr := os.Args[2]
	plcPortNum, err := strconv.Atoi(plcPortStr)
	if err != nil {
		log.Fatalf("Invalid port: %v", err)
	}

	fmt.Printf("=== Omron FINS TCP Client Test ===\n")
	fmt.Printf("PLC IP: %s, Port: %d\n\n", plcIP, plcPortNum)

	driver := fins.NewFinsTCPDriver()
	cfg := fins.DriverConfig{
		Protocol: "omron-fins-tcp",
		Config: map[string]interface{}{
			"plcIP":   plcIP,
			"plcPort": plcPortNum,
			"timeout": 3000,
		},
	}

	if err := driver.Init(cfg); err != nil {
		log.Fatalf("Failed to initialize driver: %v", err)
	}
	fmt.Println("Driver initialized")

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer driver.Disconnect()
	fmt.Println("Connected successfully")

	testCases := []struct {
		name string
		fn   func() error
	}{
		{"Test Read Points (UINT16)", testReadPointsUINT16(driver, ctx)},
		{"Test Read Points (FLOAT)", testReadPointsFLOAT(driver, ctx)},
		{"Test Read Points (BIT)", testReadPointsBIT(driver, ctx)},
		{"Test Write/Read Point (UINT16)", testWriteReadPointUINT16(driver, ctx)},
		{"Test Write/Read Point (FLOAT)", testWriteReadPointFLOAT(driver, ctx)},
		{"Test Write/Read Point (BIT)", testWriteReadPointBIT(driver, ctx)},
		{"Test Write/Read Point (INT32)", testWriteReadPointINT32(driver, ctx)},
		{"Test Write/Read Point (STRING)", testWriteReadPointSTRING(driver, ctx)},
		{"Test Health Check", testHealthCheck(driver)},
		{"Test Connection Metrics", testConnectionMetrics(driver)},
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

func testReadPointsUINT16(driver *fins.FinsTCPDriver, ctx context.Context) func() error {
	return func() error {
		points := []fins.Point{
			{ID: "D100", Address: "D100", DataType: fins.DataTypeUINT16},
			{ID: "D101", Address: "D101", DataType: fins.DataTypeUINT16},
			{ID: "D102", Address: "D102", DataType: fins.DataTypeUINT16},
		}

		results, err := driver.ReadPoints(ctx, points)
		if err != nil {
			return fmt.Errorf("ReadPoints failed: %v", err)
		}

		for _, p := range points {
			if v, ok := results[p.ID]; !ok {
				return fmt.Errorf("Point %s not found in results", p.ID)
			} else if v.Quality != fins.QualityGood {
				return fmt.Errorf("Point %s has bad quality: %v", p.ID, v.Quality)
			}
		}
		fmt.Printf("  Read 3 UINT16 points successfully\n")
		return nil
	}
}

func testReadPointsFLOAT(driver *fins.FinsTCPDriver, ctx context.Context) func() error {
	return func() error {
		points := []fins.Point{
			{ID: "D200", Address: "D200", DataType: fins.DataTypeFLOAT},
		}

		results, err := driver.ReadPoints(ctx, points)
		if err != nil {
			return fmt.Errorf("ReadPoints failed: %v", err)
		}

		val, ok := results["D200"]
		if !ok {
			return fmt.Errorf("Point D200 not found")
		}
		if val.Quality != fins.QualityGood {
			return fmt.Errorf("Point D200 has bad quality")
		}
		fmt.Printf("  Read FLOAT point: %v\n", val.Value)
		return nil
	}
}

func testReadPointsBIT(driver *fins.FinsTCPDriver, ctx context.Context) func() error {
	return func() error {
		points := []fins.Point{
			{ID: "CIO0.0", Address: "CIO0.0", DataType: fins.DataTypeBIT},
			{ID: "CIO0.1", Address: "CIO0.1", DataType: fins.DataTypeBIT},
			{ID: "CIO0.2", Address: "CIO0.2", DataType: fins.DataTypeBIT},
		}

		results, err := driver.ReadPoints(ctx, points)
		if err != nil {
			return fmt.Errorf("ReadPoints failed: %v", err)
		}

		for _, p := range points {
			if v, ok := results[p.ID]; !ok {
				return fmt.Errorf("Point %s not found", p.ID)
			} else if v.Quality != fins.QualityGood {
				return fmt.Errorf("Point %s has bad quality", p.ID)
			}
		}
		fmt.Printf("  Read 3 BIT points successfully\n")
		return nil
	}
}

func testWriteReadPointUINT16(driver *fins.FinsTCPDriver, ctx context.Context) func() error {
	return func() error {
		point := fins.Point{ID: "D300", Address: "D300", DataType: fins.DataTypeUINT16}
		testVal := uint16(54321)

		if err := driver.WritePoint(ctx, point, testVal); err != nil {
			return fmt.Errorf("WritePoint failed: %v", err)
		}

		results, err := driver.ReadPoints(ctx, []fins.Point{point})
		if err != nil {
			return fmt.Errorf("ReadPoints failed: %v", err)
		}

		if val, ok := results["D300"]; !ok {
			return fmt.Errorf("Point D300 not found")
		} else if val.Value != testVal {
			return fmt.Errorf("Value mismatch: expected %d, got %v", testVal, val.Value)
		}
		fmt.Printf("  Wrote: %d, Read: %v\n", testVal, testVal)
		return nil
	}
}

func testWriteReadPointFLOAT(driver *fins.FinsTCPDriver, ctx context.Context) func() error {
	return func() error {
		point := fins.Point{ID: "D400", Address: "D400", DataType: fins.DataTypeFLOAT}
		testVal := float32(3.14159)

		if err := driver.WritePoint(ctx, point, testVal); err != nil {
			return fmt.Errorf("WritePoint failed: %v", err)
		}

		results, err := driver.ReadPoints(ctx, []fins.Point{point})
		if err != nil {
			return fmt.Errorf("ReadPoints failed: %v", err)
		}

		if val, ok := results["D400"]; !ok {
			return fmt.Errorf("Point D400 not found")
		} else if val.Value != testVal {
			return fmt.Errorf("Value mismatch: expected %f, got %v", testVal, val.Value)
		}
		fmt.Printf("  Wrote: %f, Read: %v\n", testVal, testVal)
		return nil
	}
}

func testWriteReadPointBIT(driver *fins.FinsTCPDriver, ctx context.Context) func() error {
	return func() error {
		point := fins.Point{ID: "CIO100.0", Address: "CIO100.0", DataType: fins.DataTypeBIT}

		if err := driver.WritePoint(ctx, point, true); err != nil {
			return fmt.Errorf("WritePoint true failed: %v", err)
		}

		results, err := driver.ReadPoints(ctx, []fins.Point{point})
		if err != nil {
			return fmt.Errorf("ReadPoints failed: %v", err)
		}

		if val, ok := results["CIO100.0"]; !ok {
			return fmt.Errorf("Point CIO100.0 not found")
		} else if val.Value != true {
			return fmt.Errorf("Value mismatch: expected true, got %v", val.Value)
		}

		if err := driver.WritePoint(ctx, point, false); err != nil {
			return fmt.Errorf("WritePoint false failed: %v", err)
		}

		results, err = driver.ReadPoints(ctx, []fins.Point{point})
		if err != nil {
			return fmt.Errorf("ReadPoints failed: %v", err)
		}

		if val, ok := results["CIO100.0"]; !ok {
			return fmt.Errorf("Point CIO100.0 not found")
		} else if val.Value != false {
			return fmt.Errorf("Value mismatch: expected false, got %v", val.Value)
		}
		fmt.Printf("  Wrote: true/false, Read: %v/%v\n", true, false)
		return nil
	}
}

func testWriteReadPointINT32(driver *fins.FinsTCPDriver, ctx context.Context) func() error {
	return func() error {
		point := fins.Point{ID: "D500", Address: "D500", DataType: fins.DataTypeINT32}
		testVal := int32(-123456789)

		if err := driver.WritePoint(ctx, point, testVal); err != nil {
			return fmt.Errorf("WritePoint failed: %v", err)
		}

		results, err := driver.ReadPoints(ctx, []fins.Point{point})
		if err != nil {
			return fmt.Errorf("ReadPoints failed: %v", err)
		}

		if val, ok := results["D500"]; !ok {
			return fmt.Errorf("Point D500 not found")
		} else if val.Value != testVal {
			return fmt.Errorf("Value mismatch: expected %d, got %v", testVal, val.Value)
		}
		fmt.Printf("  Wrote: %d, Read: %v\n", testVal, testVal)
		return nil
	}
}

func testWriteReadPointSTRING(driver *fins.FinsTCPDriver, ctx context.Context) func() error {
	return func() error {
		point := fins.Point{ID: "D600", Address: "D600.20L", DataType: fins.DataTypeSTRING}
		testVal := "Hello FINS TCP"

		if err := driver.WritePoint(ctx, point, testVal); err != nil {
			return fmt.Errorf("WritePoint failed: %v", err)
		}

		results, err := driver.ReadPoints(ctx, []fins.Point{point})
		if err != nil {
			return fmt.Errorf("ReadPoints failed: %v", err)
		}

		if val, ok := results["D600"]; !ok {
			return fmt.Errorf("Point D600 not found")
		} else if val.Value != testVal {
			return fmt.Errorf("Value mismatch: expected %q, got %q", testVal, val.Value)
		}
		fmt.Printf("  Wrote: %q, Read: %q\n", testVal, testVal)
		return nil
	}
}

func testHealthCheck(driver *fins.FinsTCPDriver) func() error {
	return func() error {
		health := driver.Health()
		if health != fins.HealthStatusUp {
			return fmt.Errorf("Health status is %s, expected Up", health)
		}
		fmt.Printf("  Health: %s\n", health)
		return nil
	}
}

func testConnectionMetrics(driver *fins.FinsTCPDriver) func() error {
	return func() error {
		metrics := driver.GetConnectionMetrics()
		if !metrics.Connected {
			return fmt.Errorf("Connection is not connected")
		}
		fmt.Printf("  Connected: %v\n", metrics.Connected)
		fmt.Printf("  RemoteAddr: %s\n", metrics.RemoteAddr)
		fmt.Printf("  ReconnectCount: %d\n", metrics.ReconnectCount)
		return nil
	}
}