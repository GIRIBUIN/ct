package installer

import "strings"

func (h host) planLinux(plan *Plan, compiler, debugger bool) {
	data, err := h.readFile("/etc/os-release")
	if err != nil {
		plan.Notes = append(plan.Notes, "Cannot read /etc/os-release; automatic toolchain installation is unsupported.")
		return
	}
	family := false
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || (key != "ID" && key != "ID_LIKE") {
			continue
		}
		for _, id := range strings.Fields(strings.Trim(value, "\"'\r ")) {
			if id == "debian" || id == "ubuntu" {
				family = true
			}
		}
	}
	if !family {
		plan.Notes = append(plan.Notes, "Automatic Linux toolchain installation supports Debian/Ubuntu only. Install g++ and GDB with your distribution's documented method, then run ct doctor.")
		return
	}
	apt, err := h.lookup("apt-get")
	if err != nil {
		plan.Notes = append(plan.Notes, "apt-get is unavailable; install the toolchain manually.")
		return
	}
	command := func(args ...string) invocation { return invocation{path: apt, args: args} }
	if h.uid() != 0 {
		sudo, err := h.lookup("sudo")
		if err != nil {
			plan.Notes = append(plan.Notes, "sudo is unavailable. Ask an administrator to install the missing build-essential/gdb packages.")
			return
		}
		command = func(args ...string) invocation { return invocation{path: sudo, args: append([]string{apt}, args...)} }
	}
	packages := []string{}
	if compiler {
		packages = append(packages, "build-essential")
	}
	if debugger {
		packages = append(packages, "gdb")
	}
	plan.Actions = append(plan.Actions,
		h.commandAction("apt-update", "Refresh apt package indexes (may request sudo authentication)", command("update")),
		h.commandAction("apt-install", "Install missing packages and their required dependencies: "+strings.Join(packages, ", "), command(append([]string{"install", "--yes", "--no-upgrade"}, packages...)...), "apt-update"))
}
