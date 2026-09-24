package export

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"draftline/internal/types"
)

func validateKDPPreflight(doc Document, options types.PrintPDFOptions, spec publicationPDFSpec) error {
	if !spec.KDPReady {
		return nil
	}
	if options.IncludeCropMarks {
		return fmt.Errorf("KDP does not accept crop or trim marks. Turn Crop marks off, or turn KDP-ready checks off for another printer")
	}
	if !options.MirroredMargins {
		return fmt.Errorf("KDP print interiors need mirrored inside and outside margins. Turn Mirrored margins on")
	}
	if spec.FontSize < 7 {
		return fmt.Errorf("KDP requires interior type to be at least 7 pt")
	}
	bleed, err := strictInches("bleed", options.Bleed)
	if err != nil {
		return err
	}
	if !near(bleed, 0) && !near(bleed, 0.125) {
		return fmt.Errorf("KDP interior bleed must be either 0 or 0.125 inches")
	}
	for _, margin := range []struct{ name, value string }{
		{"gutter margin", options.GutterMargin}, {"outer margin", options.OuterMargin},
		{"top margin", options.TopMargin}, {"bottom margin", options.BottomMargin},
	} {
		if _, err := strictInches(margin.name, margin.value); err != nil {
			return err
		}
	}
	width, height := spec.TrimWidth/pointsPerInch, spec.TrimHeight/pointsPerInch
	if isHardcover(doc) {
		if doc.Edition != nil {
			paper := strings.ToLower(doc.Edition.PaperStock)
			interior := strings.ToLower(doc.Edition.Interior)
			if strings.Contains(paper, "groundwood") || (strings.Contains(interior, "standard") && containsColor(interior)) {
				return fmt.Errorf("KDP hardcover supports white or cream paper with black ink or premium color")
			}
		}
		if !hardcoverTrim(width, height) {
			return fmt.Errorf("KDP hardcover trim must be 5.5 x 8.5, 6 x 9, 6.14 x 9.21, 7 x 10, or 8.25 x 11 inches")
		}
	} else if width < 4 || width > 8.5 || height < 6 || height > 11.69 {
		return fmt.Errorf("KDP paperback trim must be 4 to 8.5 inches wide and 6 to 11.69 inches high")
	}
	return nil
}

func validateKDPPageCount(doc Document, options types.PrintPDFOptions, pages int) error {
	if options.SkipKDPChecks {
		return nil
	}
	// KDP counts leaves in pairs and rounds an odd PDF up to the next even
	// page for limits, margins and cover calculations.
	if pages%2 != 0 {
		pages++
	}
	minimum, maximum := 24, 828
	if isHardcover(doc) {
		minimum, maximum = 75, 550
	} else if doc.Edition != nil {
		paper := strings.ToLower(doc.Edition.PaperStock)
		interior := strings.ToLower(doc.Edition.Interior)
		switch {
		case strings.Contains(interior, "standard") && containsColor(interior):
			minimum, maximum = 72, 600
		case strings.Contains(paper, "cream"):
			maximum = 776
		case strings.Contains(paper, "groundwood"):
			maximum = 812
		}
	}
	if pages < minimum || pages > maximum {
		return fmt.Errorf("this %s interior has %d pages; KDP allows %d to %d for the selected format", kdpBindingName(doc), pages, minimum, maximum)
	}
	gutter, _ := strictInches("gutter margin", options.GutterMargin)
	outer, _ := strictInches("outer margin", options.OuterMargin)
	top, _ := strictInches("top margin", options.TopMargin)
	bottom, _ := strictInches("bottom margin", options.BottomMargin)
	minimumGutter := kdpGutter(pages)
	if gutter+0.000001 < minimumGutter {
		return fmt.Errorf("a %d-page KDP interior needs at least a %.3g-inch gutter; this one is %.3g inches", pages, minimumGutter, gutter)
	}
	bleed, _ := strictInches("bleed", options.Bleed)
	minimumOutside := 0.25
	if bleed > 0 {
		minimumOutside = 0.375
	}
	for _, margin := range []struct {
		name  string
		value float64
	}{{"outer", outer}, {"top", top}, {"bottom", bottom}} {
		if margin.value+0.000001 < minimumOutside {
			return fmt.Errorf("a KDP interior with this bleed needs at least %.3g-inch %s margins; this one is %.3g inches", minimumOutside, margin.name, margin.value)
		}
	}
	return nil
}

func strictInches(name, value string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < 0 {
		return 0, fmt.Errorf("%q is not a valid %s in inches", strings.TrimSpace(value), name)
	}
	return parsed, nil
}

func kdpGutter(pages int) float64 {
	switch {
	case pages <= 150:
		return 0.375
	case pages <= 300:
		return 0.5
	case pages <= 500:
		return 0.625
	case pages <= 700:
		return 0.75
	default:
		return 0.875
	}
}

func isHardcover(doc Document) bool {
	return doc.Edition != nil && strings.Contains(strings.ToLower(doc.Edition.Format), "hardcover")
}

func kdpBindingName(doc Document) string {
	if isHardcover(doc) {
		return "hardcover"
	}
	return "paperback"
}

func hardcoverTrim(width, height float64) bool {
	for _, size := range [][2]float64{{5.5, 8.5}, {6, 9}, {6.14, 9.21}, {7, 10}, {8.25, 11}} {
		if near(width, size[0]) && near(height, size[1]) {
			return true
		}
	}
	return false
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.0001 }

func containsColor(value string) bool {
	return strings.Contains(value, "color") || strings.Contains(value, "colour")
}
