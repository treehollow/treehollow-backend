package main

func addAttention(token string, pid int) (bool, error) {
	s, _, err := dbGetInfoByToken(token)
	if err != nil {
		return false, err
	}
	return addAttention2(s, token, pid)
}

func addAttention2(s string, token string, pid int) (bool, error) {
	pids := hexToIntSlice(s)
	if _, ok := containsInt(pids, pid); ok {
		return false, nil
	}
	pids = append(pids, pid)
	success, err := dbSetAttentions(token, intSliceToHex(pids))
	if success {
		//TODO: handle error here
		_, _ = plusOneAttentionIns.Exec(pid)
	}
	return success, err
}

func removeAttention(token string, pid int) (bool, error) {
	s, _, err := dbGetInfoByToken(token)
	if err != nil {
		return false, err
	}
	pids := hexToIntSlice(s)
	i, ok := containsInt(pids, pid)
	if !ok {
		return false, nil
	}
	pids = remove(pids, i)
	var success bool
	success, err = dbSetAttentions(token, intSliceToHex(pids))
	if success {
		//TODO: handle error here
		_, _ = minusOneAttentionIns.Exec(pid)
	}
	return success, err
}
