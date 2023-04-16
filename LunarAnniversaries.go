package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/abakum/MoonPhase"
)

func la(t time.Time) (years []string) {
	w0, wh := wd(t, true)
	m0 := MoonPhase.New(t)
	p0 := m0.PhaseNameLocale("")
	ph := strings.Replace(strings.ToLower(m0.PhaseNameLocale("ru")), " ", "_", -1)
	z0 := m0.ZodiacSignLocale("")
	zh := m0.ZodiacSignLocale("ru")
	f := time.Now().Year() - t.Year()
	years = append(years, fmt.Sprintf("#%s\n#%s\n#%s\n%d %s%s%s", wh, ph, zh, t.Year(), w0, p0, z0))
	fc := 0
	for i := 1; fc < 1; i++ {
		bd := t.AddDate(i, 0, 0)
		m := MoonPhase.New(bd)
		c := 0
		w, _ := wd(bd, true)
		if w0 == w {
			c++
		} else {
			w = "\u2003\u2004"
		}
		p := m.PhaseNameLocale("")
		if p0 == p {
			c++
		} else {
			p = "\u2003\u2004"
		}
		z := m.ZodiacSignLocale("")
		if z0 == z {
			c++
		} else {
			z = ""
		}
		if i > f && c > 0 || c > 1 && i < 90 {
			years = append(years, fmt.Sprintf("%d %s%s%s", bd.Year(), w, p, z))
			if i > f && c > 0 {
				fc++
			}
		}
	}
	return
}

func wd(t time.Time, iso bool) (string, string) {
	wds := []string{
		"воскресенье",
		"понедельник",
		"вторник",
		"среда",
		"четверг",
		"пятница",
		"суббота",
		"воскресенье",
	}
	isoWeekday := int(t.Weekday())
	if isoWeekday == 0 && iso {
		isoWeekday = 7
	}
	return string([]rune{rune(int('0') + isoWeekday), '\uFE0F', '\u20E3'}), wds[isoWeekday]
}
