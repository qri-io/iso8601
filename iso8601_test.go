package iso8601

import (
	"bytes"
	"encoding/json"
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

func TestIntervalString(t *testing.T) {
	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	dur, _ := ParseDuration("P1W")

	cases := []struct {
		expect string
		val    *Interval
	}{
		{"2000-01-01T00:00:00Z/2001-01-01T00:00:00Z", &Interval{Start: &start, End: &end}},
		{"2000-01-01T00:00:00Z/P1W", &Interval{Start: &start, Duration: dur}},
		{"P1W/2001-01-01T00:00:00Z", &Interval{End: &end, Duration: dur}},
		{"P1W", &Interval{Duration: dur}},
	}

	for i, c := range cases {
		got := c.val.String()
		if c.expect != got {
			t.Errorf("case %d. expected: '%s', got: '%s'", i, c.expect, got)
		}
	}
}

func TestParseRepeatingInterval(t *testing.T) {
	interval := Interval{
		Start: mustTime("2019-10-01T00:00:00Z"),
		End:   mustTime("2019-10-02T00:00:00Z"),
		Duration: Duration{
			duration: "P1D",
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

func TestRepeatingInterval(t *testing.T) {
	a, err := ParseRepeatingInterval("R1/P1W")
	if err != nil {
		t.Fatal(err)
	}
	if a.NextRep().Repititions != 0 {
		t.Errorf("expected single repitition next to return 0 repititions. got: %d", a.NextRep().Repititions)
	}

	inst := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if !a.After(inst).Equal(inst.Add(time.Hour * 24 * 7)) {
		t.Errorf("expected After to add a week")
	}

	b, err := ParseRepeatingInterval("R/P1W")
	if err != nil {
		t.Fatal(err)
	}
	if b.NextRep().Repititions != -1 {
		t.Errorf("expected unbounded repitition next to return -1 repititions. got: %d", b.NextRep().Repititions)
	}

	c, err := ParseRepeatingInterval("R/P1D/2000-01-02T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if !c.After(inst).IsZero() {
		t.Errorf("expected instant after end to return zero time")
	}

	d, err := ParseRepeatingInterval("R/2100-01-01T00:00:00Z/P1D")
	if err != nil {
		t.Fatal(err)
	}
	if !d.After(inst).IsZero() {
		t.Errorf("expected instant before start to return zero time")
	}
}

func TestRepeatingIntervalJSON(t *testing.T) {
	expect := []byte(`"R/P1W"`)
	ri := RepeatingInterval{}
	if err := json.Unmarshal(expect, &ri); err != nil {
		t.Error(err)
	}

	got, err := ri.MarshalJSON()
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(expect, got) {
		t.Errorf("byte mismatch. want: %s. got: %s", string(expect), string(got))
	}

	wrongType := []byte(`0`)
	if err := json.Unmarshal(wrongType, &ri); err == nil {
		t.Error("expected wrong json data type to error")
	}

	badString := []byte(`"wut"`)
	if err := json.Unmarshal(badString, &ri); err == nil {
		t.Error("expected bad input data to error")
	}

}

func TestRepeatingIntervalString(t *testing.T) {
	dur, _ := ParseDuration("P1W")
	ivl := Interval{Duration: dur}

	cases := []struct {
		expect string
		val    RepeatingInterval
	}{
		{"R/P1W", RepeatingInterval{Repititions: 0, Interval: ivl}},
		{"R1/P1W", RepeatingInterval{Repititions: 1, Interval: ivl}},
	}

	for i, c := range cases {
		got := c.val.String()
		if c.expect != got {
			t.Errorf("case %d. expected: '%s', got: '%s'", i, c.expect, got)
		}
	}
}

func mustTime(s string) *time.Time {
	t, err := ParseTime(s)
	if err != nil {
		panic(err)
	}
	return &t
}
