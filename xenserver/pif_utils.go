package xenserver

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// pifDataSourceModel describes the data source data model.
type pifDataSourceModel struct {
	Device     types.String `tfsdk:"device"`
	Management types.Bool   `tfsdk:"management"`
	Network    types.String `tfsdk:"network"`
}
