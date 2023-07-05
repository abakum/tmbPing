package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/abakum/MoonPhase"
	"github.com/abakum/gozodiac"
)

// list anniversaries
func la(t time.Time) (years []string) {
	const space = "\u2003\u2006" ////"\u2003\u2004"
	w0, wh := wdLocale(t, ul)
	m0 := MoonPhase.New(t)
	p0 := m0.PhaseNameLocale("")
	z0 := m0.ZodiacSignLocale("")
	s0 := gozodiac.GetZodiacSignLocale(t, "")[0]
	v0 := gozodiac.GetChineseZodiacSignLocale(t, "")
	sq := ""
	if z0 == s0 {
		sq = "²"
	}
	yl := yearLocale(ul)
	f := time.Now().Year() - t.Year()
	years = append(years, fmt.Sprintf("%s%s\n"+
		"%s%s\n"+
		"%s%s\n"+
		"%s%s_\n"+
		"%s%s\n"+
		"#%d%s %s%s%s%s%s",
		s0, hashTag(gozodiac.GetZodiacSignLocale(t, ul)[0]),
		w0, hashTag(wh),
		p0, hashTag(m0.PhaseNameLocale(ul)),
		z0, hashTag(m0.ZodiacSignLocale(ul)),
		v0, hashTag(gozodiac.GetChineseZodiacSignLocale(t, ul)),
		t.Year(), yl, w0, p0, z0, sq, v0))
	fc := 0
	for i := 1; fc < 1; i++ {
		bd := t.AddDate(i, 0, 0)
		m := MoonPhase.New(bd)
		c := 0
		w, _ := wdLocale(bd, "")
		if w0 == w {
			c++
		} else {
			w = space
		}
		p := m.PhaseNameLocale("")
		if p0 == p {
			c++
		} else {
			p = space
		}
		z := m.ZodiacSignLocale("")
		if z0 == z {
			c++
		} else {
			z = space
		}
		v := gozodiac.GetChineseZodiacSignLocale(bd, "")
		if v0 == v {
			c++
		} else {
			v = ""
		}
		if i >= f && c > 0 || c > 1 && i < 90 {
			years = append(years, fmt.Sprintf("#%d%s %s%s%s%s", bd.Year(), yl, w, p, z, v))
			if i > f && c > 0 {
				fc++
			}
		}
	}
	return
}

// weekday
func wdLocale(t time.Time, iso639_1 string) (icon string, s string) {
	type ss []string
	type ii struct {
		ss
		iso bool
	}
	ru := ss{
		"Воскресенье",
		"Понедельник",
		"Вторник",
		"Среда",
		"Четверг",
		"Пятница",
		"Суббота",
		"Воскресенье",
	}
	msIi := map[string]ii{
		"ru": {ru, true},
	}
	isoWeekday := int(t.Weekday())
	names, ok := msIi[iso639_1]
	if !ok { //en
		s = t.Weekday().String()
	} else {
		if isoWeekday == 0 && names.iso {
			isoWeekday = 7
		}
		s = names.ss[isoWeekday]
	}
	icon = string([]rune{rune(int('0') + isoWeekday), '\uFE0F', '\u20E3'})
	return
}

// year
func yearLocale(iso639_1 string) (s string) {
	msS := map[string]string{
		"ru": "г",
	}
	s, ok := msS[iso639_1]
	if !ok { //en
		s = "y"
	}
	return
}

// hash tag
func hashTag(s string) string {
	return "#" + strings.Replace(strings.ToLower(s), " ", "_", -1)
}
