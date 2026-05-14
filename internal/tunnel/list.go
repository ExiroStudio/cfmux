package tunnel

const (
	StateManaged   = "managed"
	StateUnmanaged = "unmanaged"
	StateStale     = "stale"
)

type Entry struct {
	Name   string
	UUID   string
	State  string
	Tunnel *Tunnel
}

type ListResult struct {
	Entries     []Entry
	RemoteError error
}

func List(profile string) (ListResult, error) {
	reg, err := Load(profile)
	if err != nil {
		return ListResult{}, err
	}

	remote, remoteErr := fetchRemote(profile)
	entries := merge(reg.Tunnels, remote, remoteErr != nil)

	return ListResult{Entries: entries, RemoteError: remoteErr}, nil
}

func merge(local []Tunnel, remote []RemoteTunnel, remoteFailed bool) []Entry {
	if remoteFailed {
		out := make([]Entry, 0, len(local))
		for i := range local {
			t := local[i]
			out = append(out, Entry{
				Name:   t.Name,
				UUID:   t.UUID,
				State:  StateManaged,
				Tunnel: &local[i],
			})
		}
		return out
	}

	remoteByUUID := make(map[string]RemoteTunnel, len(remote))
	for _, r := range remote {
		remoteByUUID[r.ID] = r
	}

	localByUUID := make(map[string]int, len(local))
	for i, t := range local {
		localByUUID[t.UUID] = i
	}

	out := make([]Entry, 0, len(local)+len(remote))

	for i := range local {
		t := local[i]
		state := StateStale
		if _, ok := remoteByUUID[t.UUID]; ok {
			state = StateManaged
		}
		out = append(out, Entry{
			Name:   t.Name,
			UUID:   t.UUID,
			State:  state,
			Tunnel: &local[i],
		})
	}

	for _, r := range remote {
		if _, managed := localByUUID[r.ID]; managed {
			continue
		}
		out = append(out, Entry{
			Name:  r.Name,
			UUID:  r.ID,
			State: StateUnmanaged,
		})
	}

	return out
}
