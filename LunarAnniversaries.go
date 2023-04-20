package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/abakum/MoonPhase"
	"github.com/abakum/gozodiac"
)

func la(t time.Time) (years []string) {
	w0, wh := wd(t, true)
	m0 := MoonPhase.New(t)
	p0 := m0.PhaseNameLocale("")
	z0 := m0.ZodiacSignLocale("")
	v0 := gozodiac.GetChineseZodiacSignLocale(t, "")
	f := time.Now().Year() - t.Year()
	years = append(years, fmt.Sprintf("%s%s\n"+
		"%s%s\n"+
		"%s%s\n"+
		"%s%s_\n"+
		"%s%s\n"+
		"#%dг %s%s%s%s",
		gozodiac.GetZodiacSignLocale(t, "")[0], hashTag(gozodiac.GetZodiacSignLocale(t, ul)[0]),
		w0, hashTag(wh),
		p0, hashTag(m0.PhaseNameLocale("ru")),
		z0, hashTag(m0.ZodiacSignLocale(ul)),
		v0, hashTag(gozodiac.GetChineseZodiacSignLocale(t, ul)),
		t.Year(), w0, p0, z0, v0))
	fc := 0
	for i := 1; fc < 1; i++ {
		bd := t.AddDate(i, 0, 0)
		m := MoonPhase.New(bd)
		c := 0
		w, _ := wd(bd, true)
		if w0 == w {
			c++
		} else {
			w = "\u2003\u2006" //"\u2003\u2004"
		}
		p := m.PhaseNameLocale("")
		if p0 == p {
			c++
		} else {
			p = "\u2003\u2006"
		}
		z := m.ZodiacSignLocale("")
		if z0 == z {
			c++
		} else {
			z = "\u2003\u2006"
		}
		v := gozodiac.GetChineseZodiacSignLocale(bd, "")
		if v0 == v {
			c++
		} else {
			v = ""
		}
		if i > f && c > 0 || c > 1 && i < 90 {
			years = append(years, fmt.Sprintf("#%dг %s%s%s%s", bd.Year(), w, p, z, v))
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
func hashTag(s string) string {
	return "#" + strings.Replace(strings.ToLower(s), " ", "_", -1)
}
