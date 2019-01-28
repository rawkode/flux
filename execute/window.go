package execute

type Window struct {
	Every  Duration
	Period Duration
	Offset Duration
}

func NewWindow(every, period, offset Duration) Window {
	// Normalize the offset to a small positive duration
	if offset < 0 {
		offset += every * ((offset / -every) + 1)
	} else if offset > every {
		offset -= every * (offset / every)
	}

	return Window{
		Every: every,
		Period: period,
		Offset: offset,
	}
}

// GetEarliestBounds returns the bounds for the earliest window bounds
// that contains the given time t.  For underlapping windows that
// do not contain time t, the window directly after time t will be returned.
func (w Window) GetEarliestBounds(t Time) Bounds {
	// translate to not-offset coordinate
	t = t.Add(-w.Offset)

	stop := t.Truncate(w.Every).Add(w.Every)

	// translate to offset coordinate
	stop = stop.Add(w.Offset)

	start := stop.Add(-w.Period)
	return Bounds{
		Start: start,
		Stop:  stop,
	}
}
	//if w.Period < w.Every {
	//	// underlapping windows
	//	d := t.Remainder(w.Every)
	//	if d >= w.Period {
	//		// t is between underlapping windows.
	//		// return the immediately following window
	//		start = start.Add(w.Every)
	//	}
	//} else if w.Period > w.Every {
	//	// Overlapping windows.
	//	// t may be in more than one window.
	//	// Return the earliest one.
	//	overlaps := (w.Period / w.Every) - 1
	//	rem := w.Period % w.Every
	//
	//	start = start.Add(-overlaps * w.Every)
	//
	//	if rem > 0 {
	//		d := t.Remainder(w.Every)
	//		if d < rem {
	//			// There is a fractional overlap, and t is in it,
	//			// so go back one more.
	//			start = start.Add(-w.Every)
	//		}
	//	}
	//}

	// translate to offset coordinate
//	start = start.Add(w.Offset)
//	return Bounds{
//		Start: start,
//		Stop:
//	}
//}


func (w Window) GetOverlappingBounds(b Bounds) []Bounds {
	if b.IsEmpty() {
		return []Bounds{}
	}

	c := (b.Duration() / w.Every) + (w.Period / w.Every)
	bs := make([]Bounds, 0, c)

	bi := w.GetEarliestBounds(b.Start)
	for bi.Start < b.Stop {
		bs = append(bs, bi)
		bi.Start = bi.Start.Add(w.Every)
		bi.Stop = bi.Stop.Add(w.Every)
	}

	return bs
}