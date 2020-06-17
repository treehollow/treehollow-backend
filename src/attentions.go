package main

func addAttention(token string, pid int) (bool, error) {
	s, _, err := getInfoByToken(token)
	if err != nil {
		return false, err
	}
	return addAttention2(s, token, pid)
}

func addAttention2(s string, token string, pid int) (bool, error) {
	pids := hexToIntSlice(s)
	if _, ok := contains(pids, pid); ok {
		return false, nil
	}
	pids = append(pids, pid)
	success, err := setAttentions(token, intSliceToHex(pids))
	if success {
		//TODO: handle error here
		_, _ = plusOneAttentionIns.Exec(pid)
	}
	return success, err
}

func removeAttention(token string, pid int) (bool, error) {
	s, _, err := getInfoByToken(token)
	if err != nil {
		return false, err
	}
	pids := hexToIntSlice(s)
	i, ok := contains(pids, pid)
	if !ok {
		return false, nil
	}
	pids = remove(pids, i)
	var success bool
	success, err = setAttentions(token, intSliceToHex(pids))
	if success {
		//TODO: handle error here
		_, _ = minusOneAttentionIns.Exec(pid)
	}
	return success, err
}
