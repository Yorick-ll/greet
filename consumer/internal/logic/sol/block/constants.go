package block

import (
	"errors"
	"greet/pkg/constants"
)

const ProgramStrToken = constants.ProgramStrToken
const ProgramStrToken2022 = constants.ProgramStrToken2022
const TokenStrWrapSol = constants.TokenStrWrapSol
const TokenStrUSDC = constants.TokenStrUSDC
const TokenStrUSDT = constants.TokenStrUSDT
const SolChainId = constants.SolChainId

const SolBaseAddr = constants.SolBaseTokenAddr

var ErrTokenAmountIsZero = errors.New("tokenAmount is zero,")
var ErrNotSupportWarp = errors.New("not support swap")
var ErrNotSupportInstruction = errors.New("not support instruction")
