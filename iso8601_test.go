package iso8601

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	good := []struct {
		str string
		exp time.Duration
	}{
		{"P1H", time.Hour},
		{"P1D", time.Hour * 24},
		{"P1W", time.Hour * 24 * 7},
		{"P3Y6M4DT12H30M5S", 771510605000000000},
	}
	for i, c := range good {
		got, err := ParseDuration(c.str)
		if err != nil {
			t.Errorf("case %d '%s' parse error: %s", i, c.str, err)
			continue
		}
		if got.Duration != c.exp {
			t.Errorf("case %d '%s' mismatch. expected: %d, got: %d", i, c.str, c.exp, got.Duration)
		}
	}

	bad := []struct {
		str, err string
	}{
		{"", "string '' is too short"},
		{"1D4H", "missing leading 'P' duration designator"},
		{"P1W17Y", "time units out of order: year before week"},
		{"P25Z", "unrecognized duration character 'Z'"},
		{"P99999999999999999999999999999999999999999999999999999999W", `strconv.Atoi: parsing "99999999999999999999999999999999999999999999999999999999": value out of range`},
	}

	for i, c := range bad {
		_, err := ParseDuration(c.str)
		if err == nil {
			t.Errorf("case %d '%s' produced no error", i, c.str)
			continue
		}
		if err.Error() != c.err {
			t.Errorf("case %d '%s' error mismatch. expected:\n'%s'\ngot:\n'%s'", i, c.str, c.err, err)
		}
	}
}

func TestParseInterval(t *testing.T) {
	a := mustTime("2019-10-01T00:00:00Z")
	b := mustTime("2019-10-02T00:00:00Z")
	d := Duration{
		Duration: time.Hour * 24,
	}

	good := []struct {
		str string
		exp Interval
	}{
		{"P1D", Interval{nil, nil, d}},
		{"2019-10-01T00:00:00Z/P1D", Interval{a, nil, d}},
		{"P1D/2019-10-01T00:00:00Z", Interval{nil, a, d}},
		{"2019-10-01T00:00:00Z/2019-10-02T00:00:00Z", Interval{a, b, d}},
	}

	for i, c := range good {
		got, err := ParseInterval(c.str)
		if err != nil {
			t.Errorf("case %d '%s' parse error: %s", i, c.str, err)
			continue
		}

		if c.exp.Duration.Duration != got.Duration.Duration {
			t.Errorf("case %d '%s' duration.Duration mismatch. expected: %v, got: %v", i, c.str, c.exp.Duration.Duration, got.Duration.Duration)
		}
	}

	bad := []struct {
		str, err string
	}{
		{"", "string '' is too short"},
		{"///", "too many interval designators (slashes)"},
		{"/2019-10-01T00:00:00Z", "parsing start: string '' is too short"},
		{"2019-10-01T00:00:00Z/", "parsing end: string '' is too short"},

		{"2019-13-01T00:00:00Z/", `parsing start datestamp: parsing time "2019-13-01T00:00:00Z": month out of range`},
		{"P1W/2019-13-01T00:00:00Z", `parsing end datestamp: parsing time "2019-13-01T00:00:00Z": month out of range`},

		{"Pfoo/2019-10-01T00:00:00Z", `parsing start duration: strconv.Atoi: parsing "": invalid syntax`},
		{"2019-10-01T00:00:00Z/Pfoo", `parsing end duration: strconv.Atoi: parsing "": invalid syntax`},
	}

	for i, c := range bad {
		_, err := ParseInterval(c.str)
		if err == nil {
			t.Errorf("case %d '%s' didn't error", i, c.str)
			continue
		}
		if c.err != err.Error() {
			t.Errorf("case %d '%s' error mismatch. expected:\n'%s'\ngot:\n'%s'", i, c.str, c.err, err)
		}
	}
}

func TestParseRepeatingInterval(t *testing.T) {
	interval := Interval{
		Start: mustTime("2019-10-01T00:00:00Z"),
		End:   mustTime("2019-10-02T00:00:00Z"),
		Duration: Duration{
			String:   "P1D",
			Duration: time.Hour * 24,
		},
	}

	good := []struct {
		str string
		exp RepeatingInterval
	}{
		{"R/2019-10-01T00:00:00Z/2019-10-02T00:00:00Z", RepeatingInterval{Repititions: -1, Interval: interval}},
		{"R10/2019-10-01T00:00:00Z/2019-10-02T00:00:00Z", RepeatingInterval{Repititions: 10, Interval: interval}},
		{"R02/2019-10-01T00:00:00Z/2019-10-02T00:00:00Z", RepeatingInterval{Repititions: 2, Interval: interval}},
	}

	for i, c := range good {
		got, err := ParseRepeatingInterval(c.str)
		if err != nil {
			t.Errorf("case %d '%s' parse error: %s", i, c.str, err)
			continue
		}

		if c.exp.Repititions != got.Repititions {
			t.Errorf("case %d '%s' Repitition mismatch. expected: %d got: %d", i, c.str, c.exp.Repititions, got.Repititions)
		}
	}

	bad := []struct {
		str, err string
	}{
		{"R/", "string 'R/' is too short"},
		{"R01/", "parsing interval: string '' is too short"},
		{"01/", "missing leading 'R' repeating designator"},
		{"Rfoo/", "unrecognized repeating interval character 'f'"},
		{"R99999999999999999999999999999999999999999999999999999999/P1W", `strconv.Atoi: parsing "99999999999999999999999999999999999999999999999999999999": value out of range`},
	}

	for i, c := range bad {
		_, err := ParseRepeatingInterval(c.str)
		if err == nil {
			t.Errorf("case %d '%s' didn't error", i, c.str)
			continue
		}
		if c.err != err.Error() {
			t.Errorf("case %d '%s' error mismatch. expected:\n'%s'\ngot:\n'%s'", i, c.str, c.err, err)
		}
	}
}

func TestRepeatingIntervalNext(t *testing.T) {
	a, err := ParseRepeatingInterval("R1/P1W")
	if err != nil {
		t.Fatal(err)
	}
	if a.Next().Repititions != 0 {
		t.Errorf("expected single repitition next to return 0 repititions. got: %d", a.Next().Repititions)
	}

	b, err := ParseRepeatingInterval("R/P1W")
	if err != nil {
		t.Fatal(err)
	}
	if b.Next().Repititions != -1 {
		t.Errorf("expected unbounded repitition next to return -1 repititions. got: %d", b.Next().Repititions)
	}
}

func mustTime(s string) *time.Time {
	t, err := ParseTime(s)
	if err != nil {
		panic(err)
	}
	return &t
}
