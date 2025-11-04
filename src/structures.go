package main

// Errors

type ErrorResults struct {
	Error string `json:"Error"`
}

// Blockchain

type BlockchainStats struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		AdjustedTime              int    `json:"adjusted_time"`
		AltBlocksCount            int    `json:"alt_blocks_count"`
		BlockSizeLimit            int    `json:"block_size_limit"`
		BlockSizeMedian           int    `json:"block_size_median"`
		BlockWeightLimit          int    `json:"block_weight_limit"`
		BlockWeightMedian         int    `json:"block_weight_median"`
		BootstrapDaemonAddress    string `json:"bootstrap_daemon_address"`
		BusySyncing               bool   `json:"busy_syncing"`
		Credits                   int    `json:"credits"`
		CumulativeDifficulty      int64  `json:"cumulative_difficulty"`
		CumulativeDifficultyTop64 int    `json:"cumulative_difficulty_top64"`
		DatabaseSize              int64  `json:"database_size"`
		Difficulty                int64  `json:"difficulty"`
		DifficultyTop64           int    `json:"difficulty_top64"`
		FreeSpace                 uint64 `json:"free_space"`
		GreyPeerlistSize          int    `json:"grey_peerlist_size"`
		Height                    int    `json:"height"`
		HeightWithoutBootstrap    int    `json:"height_without_bootstrap"`
		IncomingConnectionsCount  int    `json:"incoming_connections_count"`
		Mainnet                   bool   `json:"mainnet"`
		Nettype                   string `json:"nettype"`
		Offline                   bool   `json:"offline"`
		OutgoingConnectionsCount  int    `json:"outgoing_connections_count"`
		RPCConnectionsCount       int    `json:"rpc_connections_count"`
		Stagenet                  bool   `json:"stagenet"`
		StartTime                 int    `json:"start_time"`
		Status                    string `json:"status"`
		Synchronized              bool   `json:"synchronized"`
		Target                    int    `json:"target"`
		TargetHeight              int    `json:"target_height"`
		Testnet                   bool   `json:"testnet"`
		TopBlockHash              string `json:"top_block_hash"`
		TopHash                   string `json:"top_hash"`
		TxCount                   int    `json:"tx_count"`
		TxPoolSize                int    `json:"tx_pool_size"`
		Untrusted                 bool   `json:"untrusted"`
		UpdateAvailable           bool   `json:"update_available"`
		Version                   string `json:"version"`
		WasBootstrapEverUsed      bool   `json:"was_bootstrap_ever_used"`
		WhitePeerlistSize         int    `json:"white_peerlist_size"`
		WideCumulativeDifficulty  string `json:"wide_cumulative_difficulty"`
		WideDifficulty            string `json:"wide_difficulty"`
	} `json:"result"`
}

type BlockchainBlock struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Blob        string `json:"blob"`
		BlockHeader struct {
			BlockSize                 int    `json:"block_size"`
			BlockWeight               int    `json:"block_weight"`
			CumulativeDifficulty      int64  `json:"cumulative_difficulty"`
			CumulativeDifficultyTop64 int    `json:"cumulative_difficulty_top64"`
			Depth                     int    `json:"depth"`
			Difficulty                int    `json:"difficulty"`
			DifficultyTop64           int    `json:"difficulty_top64"`
			Hash                      string `json:"hash"`
			Height                    int    `json:"height"`
			LongTermWeight            int    `json:"long_term_weight"`
			MajorVersion              int    `json:"major_version"`
			MinerTxHash               string `json:"miner_tx_hash"`
			MinorVersion              int    `json:"minor_version"`
			Nonce                     int    `json:"nonce"`
			NumTxes                   int    `json:"num_txes"`
			OrphanStatus              bool   `json:"orphan_status"`
			PowHash                   string `json:"pow_hash"`
			PrevHash                  string `json:"prev_hash"`
			Reward                    int64  `json:"reward"`
			Timestamp                 int    `json:"timestamp"`
			WideCumulativeDifficulty  string `json:"wide_cumulative_difficulty"`
			WideDifficulty            string `json:"wide_difficulty"`
		} `json:"block_header"`
		Credits     int    `json:"credits"`
		JSON        string `json:"json"`
		MinerTxHash string `json:"miner_tx_hash"`
		Status      string `json:"status"`
		TopHash     string `json:"top_hash"`
		Untrusted   bool   `json:"untrusted"`
	} `json:"result"`
}

type BlockchainBlockJson struct {
	MajorVersion int    `json:"major_version"`
	MinorVersion int    `json:"minor_version"`
	Timestamp    int    `json:"timestamp"`
	PrevID       string `json:"prev_id"`
	Nonce        int    `json:"nonce"`
	MinerTx      struct {
		Version    int `json:"version"`
		UnlockTime int `json:"unlock_time"`
		Vin        []struct {
			Gen struct {
				Height int `json:"height"`
			} `json:"gen"`
		} `json:"vin"`
		Vout []struct {
			Amount int64 `json:"amount"`
			Target struct {
				Key string `json:"key"`
			} `json:"target"`
		} `json:"vout"`
		Extra         []int `json:"extra"`
		RctSignatures struct {
			Type int `json:"type"`
		} `json:"rct_signatures"`
	} `json:"miner_tx"`
	TxHashes []string `json:"tx_hashes"`
}

type CheckTxKey struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Confirmations int   `json:"confirmations"`
		InPool        bool  `json:"in_pool"`
		Received      int64 `json:"received"`
	} `json:"result"`
}

type CheckTxProof struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Confirmations int   `json:"confirmations"`
		Good          bool  `json:"good"`
		InPool        bool  `json:"in_pool"`
		Received      int64 `json:"received"`
	} `json:"result"`
}

type CheckReserveProof struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Good  bool  `json:"good"`
		Spent int   `json:"spent"`
		Total int64 `json:"total"`
	} `json:"result"`
}

type CreateIntegratedAddress struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		IntegratedAddress string `json:"integrated_address"`
		PaymentID         string `json:"payment_id"`
	} `json:"result"`
}

type TxData struct {
	Credits int    `json:"credits"`
	Status  string `json:"status"`
	TopHash string `json:"top_hash"`
	Txs     []struct {
		AsHex           string `json:"as_hex"`
		AsJSON          string `json:"as_json"`
		BlockHeight     int    `json:"block_height"`
		BlockTimestamp  int    `json:"block_timestamp"`
		DoubleSpendSeen bool   `json:"double_spend_seen"`
		InPool          bool   `json:"in_pool"`
		OutputIndices   []int  `json:"output_indices"`
		PrunableAsHex   string `json:"prunable_as_hex"`
		PrunableHash    string `json:"prunable_hash"`
		PrunedAsHex     string `json:"pruned_as_hex"`
		TxHash          string `json:"tx_hash"`
	} `json:"txs"`
	TxsAsHex  []string `json:"txs_as_hex"`
	Untrusted bool     `json:"untrusted"`
}

type CurrentBlockHeight struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Count     int    `json:"count"`
		Status    string `json:"status"`
		Untrusted string `json:"untrusted"`
	} `json:"result"`
}

type BlockHeaderRange struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Credits int `json:"credits"`
		Headers []struct {
			BlockSize                 int    `json:"block_size"`
			BlockWeight               int    `json:"block_weight"`
			CumulativeDifficulty      int64  `json:"cumulative_difficulty"`
			CumulativeDifficultyTop64 int    `json:"cumulative_difficulty_top64"`
			Depth                     int    `json:"depth"`
			Difficulty                int64  `json:"difficulty"`
			DifficultyTop64           int    `json:"difficulty_top64"`
			Hash                      string `json:"hash"`
			Height                    int    `json:"height"`
			LongTermWeight            int    `json:"long_term_weight"`
			MajorVersion              int    `json:"major_version"`
			MinerTxHash               string `json:"miner_tx_hash"`
			MinorVersion              int    `json:"minor_version"`
			Nonce                     int64  `json:"nonce"`
			NumTxes                   int    `json:"num_txes"`
			OrphanStatus              bool   `json:"orphan_status"`
			PowHash                   string `json:"pow_hash"`
			PrevHash                  string `json:"prev_hash"`
			Reward                    int64  `json:"reward"`
			Timestamp                 int    `json:"timestamp"`
			WideCumulativeDifficulty  string `json:"wide_cumulative_difficulty"`
			WideDifficulty            string `json:"wide_difficulty"`
		} `json:"headers"`
		Status    string `json:"status"`
		TopHash   string `json:"top_hash"`
		Untrusted bool   `json:"untrusted"`
	} `json:"result"`
}

type ValidateAddress struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Valid bool `json:"valid"`
	} `json:"result"`
}

// API Structures

// Blockchain

type v1XcashBlockchainUnauthorizedStats struct {
	Height                 int    `json:"height"`
	Hash                   string `json:"hash"`
	Reward                 int64  `json:"reward"`
	Size                   int64  `json:"size"`
	Version                int    `json:"version"`
	VersionBlockHeight     int    `json:"versionBlockHeight"`
	NextVersionBlockHeight int    `json:"nextVersionBlockHeight"`
	TotalPublicTx          int    `json:"totalPublicTx"`
	TotalPrivateTx         int    `json:"totalPrivateTx"`
	CirculatingSupply      int64  `json:"circulatingSupply"`
	GeneratedSupply        int64  `json:"generatedSupply"`
	TotalSupply            int64  `json:"totalSupply"`
	EmissionReward         int64  `json:"emissionReward"`
	EmissionHeight         int    `json:"emissionHeight"`
	EmissionTime           int    `json:"emissionTime"`
	InflationHeight        int    `json:"inflationHeight"`
	InflationTime          int    `json:"inflationTime"`
}

type v1XcashBlockchainUnauthorizedBlocksBlockHeight struct {
	Height       int      `json:"height"`
	Hash         string   `json:"hash"`
	Reward       int64    `json:"reward"`
	Time         int      `json:"time"`
	XcashDPOPS   bool     `json:"xcashDPOPS"`
	DelegateName string   `json:"delegateName"`
	Tx           []string `json:"tx"`
}

type v1XcashBlockchainUnauthorizedTxProve struct {
	Valid  bool  `json:"valid"`
	Amount int64 `json:"amount"`
}

type v1XcashBlockchainUnauthorizedAddressProve struct {
	Amount int64 `json:"amount"`
}

type v1XcashBlockchainUnauthorizedAddressValidate struct {
	Valid bool `json:"valid"`
}

type v1XcashBlockchainUnauthorizedAddressHistory struct {
	Tx       string `json:"tx"`
	Key      string `json:"key"`
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Amount   int64  `json:"amount"`
	Height   int    `json:"height"`
	Time     int    `json:"time"`
}

type v1XcashBlockchainUnauthorizedAddressCreateIntegrated struct {
	IntegratedAddress string `json:"integratedAddress"`
	PaymentID         string `json:"paymentId"`
}

type v1XcashBlockchainUnauthorizedTxTxHash struct {
	Height        int    `json:"height"`
	Confirmations int    `json:"confirmations"`
	Time          int    `json:"time"`
	Type          string `json:"type"`
	Sender        string `json:"sender"`
	Receiver      string `json:"receiver"`
	Amount        int64  `json:"amount"`
}

// XCASH DPOPS

type v1XcashDpopsUnauthorizedStats struct {
	MostTotalRoundsDelegateName                   string `json:"mostTotalRoundsDelegateName"`
	MostTotalRounds                               int    `json:"mostTotalRounds"`
	BestBlockVerifierOnlinePercentageDelegateName string `json:"bestBlockVerifierOnlinePercentageDelegateName"`
	BestBlockVerifierOnlinePercentage             int    `json:"bestBlockVerifierOnlinePercentage"`
	MostBlockProducerTotalRoundsDelegateName      string `json:"mostBlockProducerTotalRoundsDelegateName"`
	MostBlockProducerTotalRounds                  int    `json:"mostBlockProducerTotalRounds"`
	TotalVotes                                    int64  `json:"totalVotes"`
	TotalVoters                                   int    `json:"totalVoters"`
	AverageVote                                   int64  `json:"averageVote"`
	VotePercentage                                int    `json:"votePercentage"`
	RoundNumber                                   int    `json:"roundNumber"`
	TotalRegisteredDelegates                      int    `json:"totalRegisteredDelegates"`
	TotalOnlineDelegates                          int    `json:"totalOnlineDelegates"`
	CurrentBlockVerifiersMaximumAmount            int    `json:"currentBlockVerifiersMaximumAmount"`
	CurrentBlockVerifiersValidAmount              int    `json:"currentBlockVerifiersValidAmount"`
}

type v1XcashDpopsUnauthorizedDelegatesBasicData struct {
	Votes                    int64  `json:"votes"`
	Voters                   int    `json:"voters"`
	IPAdress                 string `json:"IPAdress"`
	DelegateName             string `json:"delegateName"`
	SharedDelegate           bool   `json:"sharedDelegate"`
	SeedNode                 bool   `json:"seedNode"`
	Online                   bool   `json:"online"`
	Fee                      int    `json:"fee"`
	TotalRounds              int    `json:"totalRounds"`
	TotalBlockProducerRounds int    `json:"totalBlockProducerRounds"`
	OnlinePercentage         int    `json:"onlinePercentage"`
}




type v2XcashDpopsUnauthorizedDelegatesBasicData struct {
	Votes                    int64  `json:"votes"`
	Voters                   int    `json:"voters"`
	IPAdress                 string `json:"IPAdress"`
	DelegateName             string `json:"delegateName"`
	SharedDelegate           bool   `json:"sharedDelegate"`
	SeedNode                 bool   `json:"seedNode"`
	Online                   bool   `json:"online"`
	Fee                      int    `json:"fee"`
	TotalRounds              int    `json:"totalRounds"`
	TotalBlockProducerRounds int    `json:"totalBlockProducerRounds"`
	OnlinePercentage         int    `json:"onlinePercentage"`
}

type v1XcashDpopsUnauthorizedDelegatesData struct {
	Votes                    int64  `json:"votes"`
	Voters                   int    `json:"voters"`
	IPAdress                 string `json:"IPAdress"`
	DelegateName             string `json:"delegateName"`
	PublicAddress            string `json:"publicAddress"`
	About                    string `json:"about"`
	Website                  string `json:"website"`
	Team                     string `json:"team"`
	Specifications           string `json:"specifications"`
	SharedDelegate           bool   `json:"sharedDelegate"`
	SeedNode                 bool   `json:"seedNode"`
	Online                   bool   `json:"online"`
	Fee                      int    `json:"fee"`
	TotalRounds              int    `json:"totalRounds"`
	TotalBlockProducerRounds int    `json:"totalBlockProducerRounds"`
	OnlinePercentage         int    `json:"onlinePercentage"`
	Rank                     int    `json:"rank"`
}

type v1XcashDpopsUnauthorizedDelegatesRounds struct {
	TotalBlocksProduced int   `json:"totalBlocksProduced"`
	TotalBlockRewards   int64 `json:"totalBlockRewards"`
	AveragePercentage   int   `json:"averagePercentage"`
	AverageTime         int   `json:"averageTime"`
	BlocksProduced      []struct {
		BlockHeight int   `json:"blockHeight"`
		BlockReward int64 `json:"blockReward"`
		Time        int   `json:"time"`
	} `json:"blocksProduced"`
}

type BlocksProduced struct {
	BlockHeight int   `json:"blockHeight"`
	BlockReward int64 `json:"blockReward"`
	Time        int   `json:"time"`
}

type v1XcashDpopsUnauthorizedDelegatesVotes struct {
	PublicAddress string `json:"publicAddress"`
	Amount        int64  `json:"amount"`
	ReserveProof  string `json:"reserveProof"`
}

type v1XcashDpopsUnauthorizedVotes struct {
	DelegateName string `json:"delegateName"`
	Amount       int64  `json:"amount"`
}

type v1XcashDpopsUnauthorizedRounds struct {
	Delegates []string `json:"delegates"`
}

type v1XcashDpopsUnauthorizedLastBlockProducer struct {
	LastBlockProducer string `json:"lastBlockProducer"`
}

// XCASH Namespace
type v1XcashNamespaceUnauthorizedStats struct {
	TotalNamesRegisteredOrRenewed int   `json:"totalNamesRegisteredOrRenewed"`
	TotalVolume                   int64 `json:"totalVolume"`
}

type v1XcashNamespaceUnauthorizedDelegatesRegistered struct {
	DelegateName  string `json:"delegateName"`
	PublicAddress string `json:"publicAddress"`
	Amount        int64  `json:"amount"`
}

type v1XcashNamespaceUnauthorizedDelegatesDelegateName struct {
	DelegateName                  string `json:"delegateName"`
	PublicAddress                 string `json:"publicAddress"`
	Amount                        int64  `json:"amount"`
	TotalNamesRegisteredOrRenewed int    `json:"totalNamesRegisteredOrRenewed"`
	TotalVolume                   int64  `json:"totalVolume"`
}

type v1XcashNamespaceUnauthorizedNamesName struct {
	Address        string `json:"address"`
	Saddress       string `json:"saddress"`
	Paddress       string `json:"paddress"`
	Expires        int    `json:"expires"`
	DelegateName   string `json:"delegateName"`
	DelegateAmount int64  `json:"delegateAmount"`
}

type v1XcashNamespaceUnauthorizedNamesStatusName struct {
	Status bool `json:"status"`
}

type v1XcashNamespaceUnauthorizedNamesStatusAddress struct {
	Status string `json:"status"`
}

type v1XcashNamespaceUnauthorizedNamesConvertName struct {
	Address  string `json:"address"`
	Saddress string `json:"saddress"`
	Paddress string `json:"paddress"`
}

type v1XcashNamespaceUnauthorizedNamesConvertAddress struct {
	Name      string `json:"name"`
	Extension string `json:"extension"`
}

// Xpayment Twitter
type v1XpaymentTwitterUnauthorizedStats struct {
	TotalUsers                     int   `json:"totalUsers"`
	AvgTipAmount                   int   `json:"avgTipAmount"`
	TotalDeposits                  int   `json:"totalDeposits"`
	TotalWithdraws                 int   `json:"totalWithdraws"`
	TotalTipsPublic                int   `json:"totalTipsPublic"`
	TotalTipsPrivate               int   `json:"totalTipsPrivate"`
	TotalVolumeSentPublic          int64 `json:"totalVolumeSentPublic"`
	TotalVolumeSentPrivate         int64 `json:"totalVolumeSentPrivate"`
	TotalTipsLastDayPublic         int   `json:"totalTipsLastDayPublic"`
	TotalTipsLastDayPrivate        int   `json:"totalTipsLastDayPrivate"`
	TotalVolumeSentLastDayPublic   int64 `json:"totalVolumeSentLastDayPublic"`
	TotalVolumeSentLastDayPrivate  int64 `json:"totalVolumeSentLastDayPrivate"`
	TotalTipsLastHourPublic        int   `json:"totalTipsLastHourPublic"`
	TotalTipsLastHourPrivate       int   `json:"totalTipsLastHourPrivate"`
	TotalVolumeSentLastHourPublic  int64 `json:"totalVolumeSentLastHourPublic"`
	TotalVolumeSentLastHourPrivate int64 `json:"totalVolumeSentLastHourPrivate"`
}

type v1XpaymentTwitterUnauthorizedStatsperday struct {
	Time   int   `json:"time"`
	Amount int   `json:"amount"`
	Volume int64 `json:"volume"`
}

type TopTips struct {
	Username string `json:"username"`
	Tips     int    `json:"tips"`
}

type TopVolumes struct {
	Username string `json:"username"`
	Volume   int    `json:"volume"`
}

type v1XpaymentTwitterUnauthorizedTopstats struct {
	TopTips    []TopTips
	TopVolumes []TopVolumes
}

type v1XpaymentTwitterUnauthorizedRecentTips struct {
	TweetID  string `json:"tweetId"`
	FromUser string `json:"fromUser"`
	ToUser   string `json:"toUser"`
	Amount   int64  `json:"amount"`
	Time     int    `json:"time"`
	Type     string `json:"type"`
}

// Database

// XCASH DPOPS
type XcashDpopsReserveProofsCollection struct {
	ID                               string `bson:"_id"`
	PublicAddressCreatedReserveProof string `bson:"public_address_created_reserve_proof"`
	PublicAddressVotedFor            string `bson:"public_address_voted_for"`
	Total                            string `bson:"total"`
	ReserveProof                     string `bson:"reserve_proof"`
}

type XcashDpopsReserveBytesCollection struct {
	ID                   string `bson:"_id"`
	BlockHeight          string `bson:"block_height"`
	ReserveBytesDataHash string `bson:"reserve_bytes_data_hash"`
	ReserveBytes         string `bson:"reserve_bytes"`
}

type XcashDpopsDelegatesCollection struct {
	ID                             string `bson:"_id"`
	PublicAddress                  string `bson:"public_address"`
	TotalVoteCount                 string `bson:"total_vote_count"`
	IPAddress                      string `bson:"IP_address"`
	DelegateName                   string `bson:"delegate_name"`
	About                          string `bson:"about"`
	Website                        string `bson:"website"`
	Team                           string `bson:"team"`
	SharedDelegateStatus           string `bson:"shared_delegate_status"`
	DelegateFee                    string `bson:"delegate_fee"`
	ServerSpecs                    string `bson:"server_specs"`
	BlockVerifierScore             string `bson:"block_verifier_score"`
	OnlineStatus                   string `bson:"online_status"`
	BlockVerifierTotalRounds       string `bson:"block_verifier_total_rounds"`
	BlockVerifierOnlineTotalRounds string `bson:"block_verifier_online_total_rounds"`
	BlockVerifierOnlinePercentage  string `bson:"block_verifier_online_percentage"`
	BlockProducerTotalRounds       string `bson:"block_producer_total_rounds"`
	BlockProducerBlockHeights      string `bson:"block_producer_block_heights"`
	PublicKey                      string `bson:"public_key"`
}

type XcashDpopsStatisticsCollection struct {
	ID                                            string `bson:"_id"`
	Username                                      string `bson:"username"`
	MostTotalRoundsDelegateName                   string `bson:"most_total_rounds_delegate_name"`
	MostTotalRounds                               string `bson:"most_total_rounds"`
	BestBlockVerifierOnlinePercentageDelegateName string `bson:"best_block_verifier_online_percentage_delegate_name"`
	BestBlockVerifierOnlinePercentage             string `bson:"best_block_verifier_online_percentage"`
	MostBlockProducerTotalRoundsDelegateName      string `bson:"most_block_producer_total_rounds_delegate_name"`
	MostBlockProducerTotalRounds                  string `bson:"most_block_producer_total_rounds"`
}

// XCASH Namespace
type XcashDpopsRemoteDataDelegatesCollection struct {
	ID                           string `bson:"_id"`
	Name                         string `bson:"name"`
	PublicAddress                string `bson:"public_address"`
	Amount                       string `bson:"amount"`
	TotalRegisteredRenewedAmount string `bson:"total_registered_renewed_amount"`
	TotalAmount                  string `bson:"total_amount"`
}

type XcashDpopsRemoteDataCollection struct {
	ID                     string `bson:"_id"`
	Name                   string `bson:"name"`
	Address                string `bson:"address"`
	Saddress               string `bson:"saddress"`
	Paddress               string `bson:"paddress"`
	SaddressList           string `bson:"saddress_list"`
	PaddressList           string `bson:"paddress_list"`
	Website                string `bson:"website"`
	SmartContractHash      string `bson:"smart_contract_hash"`
	Timestamp              string `bson:"timestamp"`
	ReserveDelegateAddress string `bson:"reserve_delegate_address"`
	ReserveDelegateAmount  string `bson:"reserve_delegate_amount"`
	TxHash                 string `bson:"tx_hash"`
}

// API
type XcashAPIStatisticsCollection struct {
	ID      string `bson:"_id"`
	Public  string `bson:"public"`
	Private string `bson:"private"`
}

// Post request data
type v1XcashBlockchainUnauthorizedTxProvePostData struct {
	Tx      string `json:"tx"`
	Address string `json:"address"`
	Key     string `json:"key"`
}

type v1XcashBlockchainUnauthorizedAddressProvePostData struct {
	Address   string `json:"address"`
	Signature string `json:"signature"`
}

type v1XcashBlockchainUnauthorizedAddressCreateIntegratedPostData struct {
	Address   string `json:"Address"`
	PaymentID string `json:"paymentId"`
}

type v1XpaymentTwitterUnauthorizedRecentTipsPostData struct {
	Sort string `json:"sort"`
	Type string `json:"type"`
}
