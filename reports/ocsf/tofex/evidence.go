// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tofex

import (
	"strconv"
	"strings"

	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
)

// Evidences maps one OCSF evidences entry onto fex evidences, one per artifact
// the entry carries. source is the entry's path in the finding, such as
// "evidences.0"; every evidence records it, with the artifact appended, in
// PropertySource. tactic, technique and confidence are set on every evidence.
func Evidences(e ocsf.Evidences, source string, tactic *fex.AttackTactic, technique *fex.AttackTechnique, confidence fex.Confidence) []*fex.Evidence {
	var out []*fex.Evidence
	add := func(artifact string, props map[string]string, set func(*fex.Evidence)) {
		ev := &fex.Evidence{
			Tactic:     tactic,
			Technique:  technique,
			Confidence: confidence,
			Properties: props,
		}
		set(ev)
		if ev.Properties == nil {
			ev.Properties = map[string]string{}
		}
		ev.Properties[PropertySource] = source + "." + artifact
		out = append(out, ev)
	}

	if a := e.Actor; a != nil {
		if a.Process != nil {
			props := map[string]string{}
			p := process(a.Process, "actor.process", props)
			add("actor.process", props, func(ev *fex.Evidence) { ev.Details = &fex.Evidence_Process{Process: p} })
		}
		if a.User != nil {
			props := map[string]string{}
			u := user(a.User, "actor.user", props)
			add("actor.user", props, func(ev *fex.Evidence) { ev.Details = &fex.Evidence_User{User: u} })
		}
	}
	if e.Process != nil {
		props := map[string]string{}
		p := process(e.Process, "process", props)
		add("process", props, func(ev *fex.Evidence) { ev.Details = &fex.Evidence_Process{Process: p} })
	}
	if e.File != nil {
		props := map[string]string{}
		f := file(e.File, "file", props)
		add("file", props, func(ev *fex.Evidence) { ev.Details = &fex.Evidence_File{File: f} })
	}
	if e.User != nil {
		props := map[string]string{}
		u := user(e.User, "user", props)
		add("user", props, func(ev *fex.Evidence) { ev.Details = &fex.Evidence_User{User: u} })
	}
	if e.SrcEndpoint != nil || e.DstEndpoint != nil || e.ConnectionInfo != nil {
		props := map[string]string{}
		c := connection(e, props)
		add("connection", props, func(ev *fex.Evidence) { ev.Details = &fex.Evidence_Connection{Connection: c} })
	}
	if q := e.Query; q != nil {
		add("query", nil, func(ev *fex.Evidence) {
			ev.Details = &fex.Evidence_DnsRecord{DnsRecord: &fex.DnsRecord{Name: q.Hostname, Type: q.Type}}
		})
	}
	if e.Container != nil {
		props := map[string]string{}
		c := container(e.Container, props)
		add("container", props, func(ev *fex.Evidence) { ev.Details = &fex.Evidence_Container{Container: c} })
	}
	return out
}

// process maps an OCSF process. path is its OCSF path relative to the artifact,
// used to key what fex has no field for.
func process(p *ocsf.Process, path string, props map[string]string) *fex.Process {
	out := processFields(p.UID, p.Pid, p.Name, p.CmdLine, p.CreatedTime, p.File, p.User, path, props)
	if len(p.Lineage) > 0 {
		props[PropertyPrefix+path+".lineage"] = strings.Join(p.Lineage, ",")
	}
	if pp := p.ParentProcess; pp != nil {
		out.Parent = processFields(pp.UID, pp.Pid, pp.Name, pp.CmdLine, pp.CreatedTime, pp.File, pp.User,
			path+".parent_process", props)
	}
	return out
}

func processFields(uid string, pid int, name, cmdLine string, created int64, f *ocsf.File, u *ocsf.User,
	path string, props map[string]string,
) *fex.Process {
	out := &fex.Process{Cmdline: cmdLine, Pid: int64(pid)}
	if out.Cmdline == "" {
		// fex names a process by its command line; the name is the best there is
		// when the line was not captured.
		out.Cmdline = name
	}
	if f != nil {
		out.Binary = file(f, path+".file", props)
	}
	if u != nil {
		out.User = user(u, path+".user", props)
	}
	setProp(props, path+".uid", uid)
	if created > 0 {
		setProp(props, path+".created_time", strconv.FormatInt(created, 10))
	}
	return out
}

func file(f *ocsf.File, path string, props map[string]string) *fex.File {
	out := &fex.File{Path: f.Path, Size: f.Size}
	if out.Path == "" {
		out.Path = f.Name
	}
	for _, h := range f.Hashes {
		switch h.AlgorithmID {
		case ocsf.FingerprintAlgorithmMD5:
			out.Md5 = h.Value
		case ocsf.FingerprintAlgorithmSHA256:
			out.Sha256 = h.Value
		default:
			alg := strings.ToLower(strings.ReplaceAll(h.Algorithm, "-", ""))
			if alg == "" {
				alg = strings.ToLower(strings.ReplaceAll(ocsf.FingerprintAlgorithmName(h.AlgorithmID), "-", ""))
			}
			if alg == "" {
				alg = strconv.Itoa(h.AlgorithmID)
			}
			setProp(props, path+".hashes."+alg, h.Value)
		}
	}
	return out
}

func user(u *ocsf.User, path string, props map[string]string) *fex.User {
	out := &fex.User{Id: u.UID, Name: u.Name}
	if u.Domain != "" {
		out.Properties = map[string]string{"domain": u.Domain}
	}
	if u.TypeID != ocsf.UserTypeUnknown {
		setProp(props, path+".type_id", strconv.Itoa(u.TypeID))
	}
	return out
}

func connection(e ocsf.Evidences, props map[string]string) *fex.Connection {
	out := &fex.Connection{}
	if src := e.SrcEndpoint; src != nil {
		out.SourceAddress = src.IP
		out.SourcePort = int32(src.Port)
		setProp(props, "src_endpoint.hostname", src.Hostname)
	}
	if dst := e.DstEndpoint; dst != nil {
		out.DestinationAddress = dst.IP
		out.DestinationPort = int32(dst.Port)
		setProp(props, "dst_endpoint.hostname", dst.Hostname)
	}
	if ci := e.ConnectionInfo; ci != nil {
		switch ci.ProtocolNum {
		case int(fex.Connection_ICMP), int(fex.Connection_TCP), int(fex.Connection_UDP):
			out.Protocol = fex.Connection_ConnectionProtocol(ci.ProtocolNum)
		default:
			if ci.ProtocolNum > 0 {
				setProp(props, "connection_info.protocol_num", strconv.Itoa(ci.ProtocolNum))
			}
		}
		setProp(props, "connection_info.protocol_name", ci.ProtocolName)
		if ci.DirectionID != ocsf.NetworkConnectionInfoDirectionUnknown {
			setProp(props, "connection_info.direction_id", strconv.Itoa(ci.DirectionID))
		}
	}
	return out
}

func container(c *ocsf.Container, props map[string]string) *fex.Container {
	out := &fex.Container{Name: c.Name}
	if img := c.Image; img != nil {
		out.ImageUri = img.Name
		if img.Tag != "" && !strings.Contains(img.Name, "@") {
			out.ImageUri += ":" + img.Tag
		}
		setProp(props, "container.image.uid", img.UID)
	}
	if c.Hash != nil {
		out.Digest = c.Hash.Value
	}
	setProp(props, "container.uid", c.UID)
	return out
}

func setProp(props map[string]string, path, v string) {
	if v != "" {
		props[PropertyPrefix+path] = v
	}
}
