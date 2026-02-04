package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/stretchr/testify/suite"

	cmdcfg "github.com/evmos/evmos/v19/cmd/config"
)

type MsgsTestSuite struct {
	suite.Suite
}

func TestMsgsTestSuite(t *testing.T) {
	// Set up SDK config with nxq prefix before running tests (if not already sealed)
	config := sdk.GetConfig()
	if config.GetBech32AccountAddrPrefix() != cmdcfg.Bech32PrefixAccAddr {
		cmdcfg.SetBech32Prefixes(config)
		config.Seal()
	}
	
	suite.Run(t, new(MsgsTestSuite))
}

func (suite *MsgsTestSuite) TestMsgUpdateValidateBasic() {
	testCases := []struct {
		name      string
		msgUpdate *MsgUpdateParams
		expPass   bool
	}{
		{
			"fail - invalid authority address",
			&MsgUpdateParams{
				Authority: "invalid",
				Params:    DefaultParams(),
			},
			false,
		},
		{
			"pass - valid msg",
			&MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    DefaultParams(),
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.msgUpdate.ValidateBasic()
			if tc.expPass {
				suite.NoError(err)
			} else {
				suite.Error(err)
			}
		})
	}
}
