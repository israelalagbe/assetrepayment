package domain

const (
	AssetValueKobo int64 = 1_000_000_00 // 1,000,000 NGN in kobo

	// RepaymentTermWeeks is the number of weeks over which the asset is repaid.
	RepaymentTermWeeks int = 50
)

type Customer struct {
	ID              string
	OutstandingKobo int64
	TotalPaidKobo   int64
	CreatedAt       string
}
