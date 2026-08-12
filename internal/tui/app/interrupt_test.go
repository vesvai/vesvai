package app

import (
	"testing"
	"time"
)

func TestInterruptGateFirstPressArmsSecondPressFires(t *testing.T) {
	g := interruptGate{window: time.Second}
	now := time.Now()

	if g.armed() {
		t.Fatal("gate must start unarmed")
	}
	if g.press(now) {
		t.Fatal("first press must only arm the gate")
	}
	if !g.armed() {
		t.Fatal("gate must be armed after first press")
	}
	if !g.press(now.Add(500 * time.Millisecond)) {
		t.Fatal("second press within the window must fire the interrupt")
	}
	if g.armed() {
		t.Fatal("gate must disarm after firing")
	}
}

func TestInterruptGateArmExpires(t *testing.T) {
	g := interruptGate{window: time.Second}
	now := time.Now()

	g.press(now)
	if !g.armed() {
		t.Fatal("gate should be armed")
	}
	g.expire(now.Add(2 * time.Second))
	if g.armed() {
		t.Fatal("arm must expire after the window")
	}
	if g.press(now.Add(3 * time.Second)) {
		t.Fatal("press after expiry must not fire")
	}
	if !g.armed() {
		t.Fatal("press after expiry must re-arm")
	}
}

func TestInterruptGateSecondPressAfterWindowRearms(t *testing.T) {
	g := interruptGate{window: time.Second}
	now := time.Now()

	g.press(now)
	if g.press(now.Add(2 * time.Second)) {
		t.Fatal("press outside the window must not fire")
	}
	if !g.press(now.Add(2*time.Second + 400*time.Millisecond)) {
		t.Fatal("press within the fresh window must fire")
	}
}
