package components

import "testing"

func TestLogoGeometry(t *testing.T) {
	lg := DefaultLogo()
	if lg.Width != 33 {
		t.Errorf("DefaultLogo width = %d, want 33", lg.Width)
	}
	if lg.Height != 5 {
		t.Errorf("DefaultLogo height = %d, want 5", lg.Height)
	}
}

func TestLogoBlinkToggles(t *testing.T) {
	lg := DefaultLogo()
	lg.SetBlink(false)
	lg.OnTick(true)
	if !lg.blinkOn {
		t.Error("OnTick(true) should enable blink")
	}
}
