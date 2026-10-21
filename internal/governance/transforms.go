package governance

func (sess *Session) transforms() []Transform {
	return []Transform{orderFindings}
}
