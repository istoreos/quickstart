package service

type StaticAssignmentWriteInput struct {
	Action             string
	AssignedMAC        string
	AssignedIP         string
	BindIP             bool
	Hostname           string
	TagName            string
	TagTitle           string
	MaterializeAutoTag bool
}

func (input StaticAssignmentWriteInput) IsNoOpAdd() bool {
	return input.Action == "add" &&
		input.TagName == "" &&
		input.TagTitle == "" &&
		input.Hostname == "" &&
		input.AssignedIP == ""
}
