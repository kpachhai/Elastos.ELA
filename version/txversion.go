package version

import (
	"errors"
	"math"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

type TxVersion interface {
	GetVersion() byte

	CheckOutputPayload(output *core.Output) error
	CheckOutputProgramHash(programHash Uint168) error
	CheckCoinbaseMinerReward(tx *core.Transaction, totalReward Fixed64) error
	CheckCoinbaseArbitratorsReward(coinbase *core.Transaction, rewardInCoinbase Fixed64) error
	CheckVoteProducerOutputs(outputs []*core.Output, references map[*core.Input]*core.Output) error
}

type TxVersionMain struct {
}

func (v *TxVersionMain) GetVersion() byte {
	return 9
}

func (v *TxVersionMain) CheckOutputPayload(output *core.Output) error {
	if err := output.OutputPayload.Validate(); err != nil {
		return err
	} else {
		return nil
	}
}

func (v *TxVersionMain) CheckOutputProgramHash(programHash Uint168) error {
	var empty = Uint168{}
	prefix := programHash[0]
	if prefix == PrefixStandard ||
		prefix == PrefixMultisig ||
		prefix == PrefixCrossChain ||
		programHash == empty {
		return nil
	}
	return errors.New("Invalid program hash prefix.")
}

func (v *TxVersionMain) CheckCoinbaseMinerReward(tx *core.Transaction, totalReward Fixed64) error {
	minerReward := tx.Outputs[1].Value
	if Fixed64(minerReward) < Fixed64(float64(totalReward)*0.35) {
		return errors.New("Reward to dpos in coinbase < 35%")
	}

	return nil
}

func (v *TxVersionMain) CheckCoinbaseArbitratorsReward(coinbase *core.Transaction, rewardInCoinbase Fixed64) error {
	outputAddressMap := make(map[Uint168]Fixed64)

	for i := 2; i < len(coinbase.Outputs); i++ {
		outputAddressMap[coinbase.Outputs[i].ProgramHash] = coinbase.Outputs[i].Value
	}

	arbitratorsHashes := blockchain.DefaultLedger.Arbitrators.GetArbitratorsProgramHashes()
	candidatesHashes := blockchain.DefaultLedger.Arbitrators.GetCandidatesProgramHashes()
	if len(arbitratorsHashes)+len(candidatesHashes) != len(coinbase.Outputs)-2 {
		return errors.New("Coinbase output count not match.")
	}

	dposTotalReward := Fixed64(float64(rewardInCoinbase) * 0.35)
	totalBlockConfirmReward := float64(dposTotalReward) * 0.25
	totalTopProducersReward := float64(dposTotalReward) * 0.75
	individualBlockConfirmReward := Fixed64(math.Floor(totalBlockConfirmReward / float64(len(arbitratorsHashes))))
	individualProducerReward := Fixed64(math.Floor(totalTopProducersReward / float64(len(arbitratorsHashes)+len(candidatesHashes))))

	for _, v := range arbitratorsHashes {

		amount, ok := outputAddressMap[*v]
		if !ok {
			return errors.New("Unknown dpos reward address.")
		}

		if amount != individualProducerReward+individualBlockConfirmReward {
			return errors.New("Incorrect dpos reward amount.")
		}
	}

	for _, v := range candidatesHashes {

		amount, ok := outputAddressMap[*v]
		if !ok {
			return errors.New("Unknown dpos reward address.")
		}

		if amount != individualProducerReward+individualBlockConfirmReward {
			return errors.New("Incorrect dpos reward amount.")
		}
	}

	return nil
}

func (v *TxVersionMain) CheckVoteProducerOutputs(outputs []*core.Output, references map[*core.Input]*core.Output) error {
	programHashes := make(map[Uint168]struct{})
	for _, v := range references {
		programHashes[v.ProgramHash] = struct{}{}
	}

	for _, o := range outputs {
		if o.OutputType == core.VoteOutput {
			if _, ok := programHashes[o.ProgramHash]; !ok {
				return errors.New("Invalid vote output")
			}
		}
	}

	return nil
}