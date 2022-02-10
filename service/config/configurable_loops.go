package config

type ConfigurableLoop string

const (
	ConfigurableLoopMinting            ConfigurableLoop = "minting"
	ConfigurableLoopPackContractEvents ConfigurableLoop = "packContractEvents"

	ConfigurableLoopSentTransactions     ConfigurableLoop = "sentTransactions"
	ConfigurableLoopSendableTransactions ConfigurableLoop = "sendableTransactions"
)
