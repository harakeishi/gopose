package presenter

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/harakeishi/gopose/pkg/types"
)

// TablePresenter formats CLI output as aligned plain-text tables.
type TablePresenter struct {
	w io.Writer
}

// NewTablePresenter creates a new TablePresenter writing to w.
func NewTablePresenter(w io.Writer) *TablePresenter {
	return &TablePresenter{w: w}
}

func (p *TablePresenter) Progress(message string) {
	fmt.Fprintln(p.w, message)
}

func (p *TablePresenter) PortConflicts(conflicts []types.PortConflictInfo) {
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Port Conflicts:")

	var resolved []types.PortConflictInfo
	for _, c := range conflicts {
		if c.Resolution != nil {
			resolved = append(resolved, c)
		}
	}

	if len(resolved) == 0 {
		fmt.Fprintln(p.w, "  (none)")
		return
	}

	tw := tabwriter.NewWriter(p.w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "  SERVICE\tFROM\tTO")
	for _, c := range resolved {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n",
			c.ServiceName,
			strconv.Itoa(c.Port),
			strconv.Itoa(c.Resolution.ResolvedPort),
		)
	}
	tw.Flush()
}

func (p *TablePresenter) NetworkConflicts(conflicts []types.NetworkConflictInfo) {
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Network Conflicts:")

	var resolved []types.NetworkConflictInfo
	for _, c := range conflicts {
		if c.Resolution != nil {
			resolved = append(resolved, c)
		}
	}

	if len(resolved) == 0 {
		fmt.Fprintln(p.w, "  (none)")
		return
	}

	tw := tabwriter.NewWriter(p.w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "  NETWORK\tFROM\tTO")
	for _, c := range resolved {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n",
			c.NetworkName,
			c.OriginalSubnet,
			c.Resolution.ResolvedSubnet,
		)
	}
	tw.Flush()
}

func (p *TablePresenter) Result(message string) {
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, message)
}
