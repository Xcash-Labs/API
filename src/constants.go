package main

const XCASH_PROOF_OF_STAKE_BLOCK_HEIGHT = 2
const BLOCK_VERIFIERS_AMOUNT = 50
const BLOCK_VERIFIERS_VALID_AMOUNT = 27
const BLOCKCHAIN_DIRECTORY = "/root/.X-CASH/"
const XCASH_DPOPS_DATABASE = "XCASH_PROOF_OF_STAKE"
const XPAYMENT_TWITTER_DATABASE = "xpaymentTwitter"
const time_day = 86400;
const time_hour = 3600;
const START_TIME = 1654041600;
// X-Cash emission and supply constants
const xcashMoneySupply uint64 = 100_000_000_000_000
const xcashAtomicUnits uint64 = 1_000_000
const xcashEmissionSpeedFactor uint = 21
const xcashTailReward uint64 = 700_000