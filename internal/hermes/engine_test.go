package hermes

import (
	"context"
	"testing"
	"time"
)

func TestHermesEngineSingleton(t *testing.T) {
	e1 := GetEngine()
	e2 := GetEngine()

	if e1 == nil || e2 == nil {
		t.Fatal("GetEngine no debería retornar nil")
	}

	if e1 != e2 {
		t.Errorf("Se esperaba que GetEngine fuera un singleton idéntico")
	}
}

func TestHermesEngineLifecycle(t *testing.T) {
	engine := GetEngine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx, 100*time.Millisecond)

	if !engine.isRunning {
		t.Errorf("Se esperaba que engine.isRunning fuera true tras Start()")
	}

	// Detener
	engine.Stop()

	if engine.isRunning {
		t.Errorf("Se esperaba que engine.isRunning fuera false tras Stop()")
	}
}
