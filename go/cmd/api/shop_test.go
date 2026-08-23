package main

import (
	"os"
	"testing"
)

func TestStringSetTrimsAndDropsEmpties(t *testing.T) {
	set := stringSet([]string{" tshirt.adult ", "", "mug ", "   "})
	if len(set) != 2 {
		t.Fatalf("stringSet size = %d, want 2 (%v)", len(set), set)
	}
	if !set["tshirt.adult"] || !set["mug"] {
		t.Errorf("stringSet = %v, want trimmed tshirt.adult and mug", set)
	}
}

func TestClosedSKUsFromEnv(t *testing.T) {
	tests := []struct {
		name string
		// set reports whether the variable exists at all; an unset variable and
		// one set to "" must mean different things.
		set   bool
		value string
		want  map[string]bool
	}{
		{
			name: "unset falls back to the default closed set",
			set:  false,
			want: map[string]bool{"tshirt.adult": true},
		},
		{
			// The escape hatch: re-opening the sale without a code change. This
			// is why closedSKUsFromEnv reads the environment directly instead of
			// using getEnvAsSlice, which returns its default for an empty value.
			name:  "set but empty closes nothing",
			set:   true,
			value: "",
			want:  map[string]bool{},
		},
		{
			name:  "a list closes exactly those SKUs",
			set:   true,
			value: "tshirt.adult,mug.enamel",
			want:  map[string]bool{"tshirt.adult": true, "mug.enamel": true},
		},
		{
			name:  "whitespace and trailing separators are tolerated",
			set:   true,
			value: " tshirt.adult , ",
			want:  map[string]bool{"tshirt.adult": true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(closedProductsEnv, tc.value)
			} else {
				// t.Setenv cannot unset, and the variable may well be present in
				// the developer's environment, so restore it by hand.
				orig, had := os.LookupEnv(closedProductsEnv)
				os.Unsetenv(closedProductsEnv)
				t.Cleanup(func() {
					if had {
						os.Setenv(closedProductsEnv, orig)
					}
				})
			}
			got := closedSKUsFromEnv()
			if len(got) != len(tc.want) {
				t.Fatalf("closedSKUsFromEnv() = %v, want %v", got, tc.want)
			}
			for sku := range tc.want {
				if !got[sku] {
					t.Errorf("closedSKUsFromEnv() = %v, missing %q", got, sku)
				}
			}
		})
	}
}

func TestSkuClosed(t *testing.T) {
	app := &application{}
	app.config.shop.closedSKUs = stringSet([]string{"tshirt.adult"})

	if !app.skuClosed("tshirt.adult") {
		t.Error("tshirt.adult should be closed")
	}
	// The point of a per-SKU switch: closing the year shirt must not close the
	// rest of the catalogue, present or future.
	for _, open := range []string{"participation.patrulje", "participation.klan", "tshirt.plain", "mug.enamel", ""} {
		if app.skuClosed(open) {
			t.Errorf("%q should be open", open)
		}
	}
}

func TestSkuClosedWithEmptySet(t *testing.T) {
	app := &application{}
	app.config.shop.closedSKUs = stringSet(nil)
	if app.skuClosed("tshirt.adult") {
		t.Error("an empty closed set must leave every product open")
	}
}

func TestClosedProductsIsSortedAndNeverNil(t *testing.T) {
	app := &application{}
	app.config.shop.closedSKUs = stringSet([]string{"mug.enamel", "tshirt.adult", "cap"})

	got := app.closedProducts()
	want := []string{"cap", "mug.enamel", "tshirt.adult"}
	if len(got) != len(want) {
		t.Fatalf("closedProducts() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("closedProducts() = %v, want %v (sorted)", got, want)
		}
	}

	app.config.shop.closedSKUs = nil
	if got := app.closedProducts(); got == nil || len(got) != 0 {
		t.Errorf("closedProducts() = %v, want an empty non-nil slice", got)
	}
}
