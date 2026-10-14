package params

import (
	"net/url"
	"testing"
)

func TestParams(t *testing.T) {
	v, _ := url.ParseQuery("name=+ex+&type=mx&count=5&bad=x&flag=true&long=abcdef")
	q := New(v)
	if q.String("name", "", true, 10) != "ex" || q.Enum("type", "A", "A", "MX") != "MX" || q.Int("count", 1, 1, 10) != 5 || !q.Bool("flag", false) {
		t.Fatal("valid values not parsed")
	}
	if q.Err() != nil {
		t.Fatal(q.Err())
	}
	for name, f := range map[string]func(*Q){
		"missing":  func(q *Q) { q.String("missing", "", true, 10) },
		"long":     func(q *Q) { q.String("long", "", false, 3) },
		"type":     func(q *Q) { q.Enum("type", "A", "A", "AAAA") },
		"count":    func(q *Q) { q.Int("count", 1, 1, 4) },
		"bad-int":  func(q *Q) { q.Int("bad", 1, 1, 4) },
		"bad-bool": func(q *Q) { q.Bool("bad", false) },
		"family":   func(q *Q) { q.v.Set("family", "5"); q.Family() },
	} {
		q := New(v)
		f(q)
		if q.Err() == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
