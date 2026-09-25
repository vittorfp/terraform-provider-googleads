package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Tiny HCL-output helpers. We only need to emit a handful of attribute kinds
// and one block shape (`resource`, `import`, `provider`, `terraform`), so a
// dependency on hashicorp/hcl/hclwrite would be overkill — strconv.Quote
// gives us valid HCL string escaping (\n \r \t \\ \" \uNNNN) for free.

type hclWriter struct {
	w   io.Writer
	err error
}

func (h *hclWriter) printf(format string, args ...any) {
	if h.err != nil {
		return
	}
	_, h.err = fmt.Fprintf(h.w, format, args...)
}

// attrString writes `<key> = "<value>"` (quoted+escaped) with the given indent.
func (h *hclWriter) attrString(indent, key, value string) {
	h.printf("%s%s = %s\n", indent, key, strconv.Quote(value))
}

func (h *hclWriter) attrInt(indent, key string, v int64) {
	h.printf("%s%s = %d\n", indent, key, v)
}

func (h *hclWriter) attrBool(indent, key string, v bool) {
	h.printf("%s%s = %t\n", indent, key, v)
}

// attrRef writes `<key> = <expr>` with no quoting — used for cross-resource
// references like `googleads_campaign_budget.budget_42.id`.
func (h *hclWriter) attrRef(indent, key, expr string) {
	h.printf("%s%s = %s\n", indent, key, expr)
}

func (h *hclWriter) attrStringList(indent, key string, vs []string) {
	if len(vs) == 0 {
		h.printf("%s%s = []\n", indent, key)
		return
	}
	var quoted []string
	for _, v := range vs {
		quoted = append(quoted, strconv.Quote(v))
	}
	h.printf("%s%s = [%s]\n", indent, key, strings.Join(quoted, ", "))
}

func (h *hclWriter) blockOpen(format string, args ...any) {
	h.printf(format+" {\n", args...)
}

func (h *hclWriter) blockClose() {
	h.printf("}\n\n")
}

func (h *hclWriter) comment(text string) {
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		h.printf("# %s\n", line)
	}
}
