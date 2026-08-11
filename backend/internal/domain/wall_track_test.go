package domain

import (
	"testing"
	"time"
)

func testBidWall(price string) OrderBookWall {
	return OrderBookWall{Side: "bid", Price: price, Quantity: "10", Notional: "1000", Share: 0.4}
}

func TestWallMemory_FirstLookIsShort(t *testing.T) {
	m := NewWallMemory()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	walls := []OrderBookWall{testBidWall("100")}
	m.Observe(now, "binance", "BTCUSDT", walls)
	if walls[0].Behavior != WallBehaviorShort || walls[0].AppearCount != 1 {
		t.Fatalf("%+v", walls[0])
	}
}

func TestWallMemory_PersistentAfterStreak(t *testing.T) {
	m := NewWallMemory()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	walls := []OrderBookWall{testBidWall("100")}
	m.Observe(now, "binance", "BTCUSDT", walls)
	later := now.Add(wallPersistentMin + time.Second)
	walls = []OrderBookWall{testBidWall("100.05")} // same zone
	m.Observe(later, "binance", "BTCUSDT", walls)
	if walls[0].Behavior != WallBehaviorPersistent {
		t.Fatalf("want persistent, got %+v", walls[0])
	}
	if walls[0].PresentForSeconds < wallPersistentMin.Seconds() {
		t.Fatalf("streak %v", walls[0].PresentForSeconds)
	}
}

func TestWallMemory_SuspiciousFlicker(t *testing.T) {
	m := NewWallMemory()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	for i := 0; i < wallSuspiciousAppears; i++ {
		if i > 0 {
			// disappear longer than grace
			m.Observe(now.Add(time.Duration(i)*20*time.Second-10*time.Second), "binance", "ETHUSDT", nil)
		}
		walls := []OrderBookWall{testBidWall("50")}
		m.Observe(now.Add(time.Duration(i)*20*time.Second), "binance", "ETHUSDT", walls)
		if i == wallSuspiciousAppears-1 && walls[0].Behavior != WallBehaviorSuspicious {
			t.Fatalf("want suspicious after %d flips, got %+v", i+1, walls[0])
		}
	}
}

func TestWallMemory_ShortGapIsNotAFlip(t *testing.T) {
	m := NewWallMemory()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	walls := []OrderBookWall{testBidWall("100")}
	m.Observe(now, "binance", "BTCUSDT", walls)
	m.Observe(now.Add(2*time.Second), "binance", "BTCUSDT", nil)
	walls = []OrderBookWall{testBidWall("100")}
	m.Observe(now.Add(5*time.Second), "binance", "BTCUSDT", walls)
	if walls[0].AppearCount != 1 {
		t.Fatalf("grace should keep one appear, got %d", walls[0].AppearCount)
	}
	if walls[0].Behavior == WallBehaviorSuspicious {
		t.Fatalf("gap must not look spoofy: %+v", walls[0])
	}
}

func TestWallMemory_DifferentSideIsOtherTrack(t *testing.T) {
	m := NewWallMemory()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	m.Observe(now, "binance", "BTCUSDT", []OrderBookWall{testBidWall("100")})
	asks := []OrderBookWall{{Side: "ask", Price: "100", Quantity: "10", Notional: "1000"}}
	m.Observe(now, "binance", "BTCUSDT", asks)
	if asks[0].AppearCount != 1 {
		t.Fatalf("ask should be its own track: %+v", asks[0])
	}
}
