package request

type UserWithdrawApplyRequest struct {
	UserId uint `json:"user_id"`
}

type UserWithdrawApprovalRequest struct {
	Id uint `json:"id"` // 记录 ID
}
