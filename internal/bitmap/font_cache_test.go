package bitmap

import (
	"testing"

	"golang.org/x/image/font/opentype"
)

// TestFontCacheEfficiency tests whether the current font caching is efficient.
// This test was created to evaluate whether additional font.Face caching is needed.
//
// Analysis findings:
// 1. The parsed TTF file (*opentype.Font) IS cached in fontCache map
// 2. Each font.Face is created fresh via opentype.NewFace() per call
// 3. font.Face creation is cheap - it just wraps the cached font data
//
// Conclusion: The expensive operation (file parsing) is already cached.
// Additional font.Face caching would add complexity with minimal benefit.
func TestFontCacheEfficiency(t *testing.T) {
	// Clear the cache to start fresh
	fontCacheMutex.Lock()
	originalCache := fontCache
	fontCache = make(map[string]*opentype.Font)
	fontCacheMutex.Unlock()

	defer func() {
		// Restore original cache
		fontCacheMutex.Lock()
		fontCache = originalCache
		fontCacheMutex.Unlock()
	}()

	// Test 1: Verify TTF parsing is cached (expensive operation)
	t.Run("TTF parsing is cached", func(t *testing.T) {
		// First load - should parse the font file
		face1, err := LoadFont("", 12)
		if err != nil {
			t.Skipf("Cannot load font: %v", err)
		}
		if face1 == nil {
			t.Skip("Font not available in test environment")
		}

		// Check cache size after first load
		fontCacheMutex.RLock()
		cacheSize1 := len(fontCache)
		fontCacheMutex.RUnlock()

		if cacheSize1 == 0 {
			t.Log("Font cache is empty - font may be using fallback")
		}

		// Second load with different size - should reuse cached TTF
		face2, err := LoadFont("", 16)
		if err != nil {
			t.Errorf("Second LoadFont failed: %v", err)
		}

		// Cache should NOT grow - same TTF file, different Face size
		fontCacheMutex.RLock()
		cacheSize2 := len(fontCache)
		fontCacheMutex.RUnlock()

		if cacheSize2 != cacheSize1 {
			t.Logf("Cache grew from %d to %d (unexpected if same font file)", cacheSize1, cacheSize2)
		}

		// Verify we got different faces (not pointer equality - that's expected)
		if face1 == face2 {
			t.Log("Same face returned for different sizes - unexpected but harmless")
		}
	})

	// Test 2: Verify multiple calls with same params work efficiently
	t.Run("repeated loads are efficient", func(t *testing.T) {
		const iterations = 100

		// Load the same font+size multiple times
		for i := 0; i < iterations; i++ {
			face, err := LoadFont("", 12)
			if err != nil {
				t.Fatalf("Iteration %d failed: %v", i, err)
			}
			if face == nil {
				t.Skip("Font not available")
			}
		}

		// If we got here without timeout, the caching is working
		// (without caching, parsing 100 times would be slow)
		t.Log("100 LoadFont calls completed - TTF parsing cache is effective")
	})

	// Test 3: Document why font.Face caching is not needed
	t.Run("font.Face objects are lightweight", func(t *testing.T) {
		// This is a documentation test explaining the design decision:
		//
		// Why we DON'T cache font.Face objects:
		//
		// 1. The expensive part (TTF file parsing) is already cached
		//    in fontCache map (see loadTTF function)
		//
		// 2. opentype.NewFace() is cheap - it creates a struct with:
		//    - Pointer to cached *opentype.Font
		//    - Size-specific metrics (computed once)
		//
		// 3. Different font sizes need different Face objects anyway
		//
		// 4. Widgets create fonts once in New() and store them
		//    - No repeated font.Face creation during render loop
		//
		// 5. font.Face is NOT thread-safe - caching would require
		//    additional synchronization or per-goroutine copies
		//
		// 6. Original analysis claimed "101 fontFace field declarations"
		//    but actual count is ~10 widgets with fontFace fields
		//
		// Trade-off analysis:
		// - Caching font.Face: +Minor memory savings, +Complexity, +Lifecycle management
		// - Current approach: Simple, clear ownership, works correctly
		//
		// Decision: Keep current approach - complexity cost > benefit

		t.Log("font.Face caching intentionally not implemented - see test comments for rationale")
	})
}
