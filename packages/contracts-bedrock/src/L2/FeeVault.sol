// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { SafeCall } from "src/libraries/SafeCall.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IL1Block } from "src/L2/interfaces/IL1Block.sol";

// Interfaces
import { IL2ToL1MessagePasser } from "src/L2/interfaces/IL2ToL1MessagePasser.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";

/// @title FeeVault
/// @notice The FeeVault contract contains the basic logic for the various different vault contracts
///         used to hold fee revenue generated by the L2 system.
abstract contract FeeVault {
    /// @notice The minimum gas limit for the FeeVault withdrawal transaction.
    uint32 internal constant WITHDRAWAL_MIN_GAS = 400_000;

    /// @notice
    function L1_BLOCK() internal pure returns (IL1Block) {
        return IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES);
    }

    /// @notice Total amount of wei processed by the contract.
    uint256 public totalProcessed;

    /// @notice Reserve extra slots in the storage layout for future upgrades.
    uint256[48] private __gap;

    /// @notice Emitted each time a withdrawal occurs. This event will be deprecated
    ///         in favor of the Withdrawal event containing the WithdrawalNetwork parameter.
    /// @param value Amount that was withdrawn (in wei).
    /// @param to    Address that the funds were sent to.
    /// @param from  Address that triggered the withdrawal.
    event Withdrawal(uint256 value, address to, address from);

    /// @notice Emitted each time a withdrawal occurs.
    /// @param value             Amount that was withdrawn (in wei).
    /// @param to                Address that the funds were sent to.
    /// @param from              Address that triggered the withdrawal.
    /// @param withdrawalNetwork Network which the to address will receive funds on.
    event Withdrawal(uint256 value, address to, address from, Types.WithdrawalNetwork withdrawalNetwork);

    /// @notice Allow the contract to receive ETH.
    receive() external payable { }

    /// @notice Returns the configuration of the FeeVault.
    function config()
        public
        view
        virtual
        returns (address recipient_, uint256 amount_, Types.WithdrawalNetwork network_);

    /// @notice Minimum balance before a withdrawal can be triggered.
    function minWithdrawalAmount() public view virtual returns (uint256 amount_) {
        (, amount_,) = config();
    }

    /// @notice Minimum balance before a withdrawal can be triggered.
    ///         Use the `minWithdrawalAmount()` getter as this is deprecated
    ///         and is subject to be removed in the future.
    /// @custom:legacy true
    function MIN_WITHDRAWAL_AMOUNT() public view returns (uint256) {
        return minWithdrawalAmount();
    }

    /// @notice Account that will receive the fees. Can be located on L1 or L2.
    function recipient() public view virtual returns (address recipient_) {
        (recipient_,,) = config();
    }

    /// @notice Account that will receive the fees. Can be located on L1 or L2.
    ///         Use the `recipient()` getter as this is deprecated
    ///         and is subject to be removed in the future.
    /// @custom:legacy
    function RECIPIENT() public view returns (address) {
        return recipient();
    }

    /// @notice Network which the recipient will receive fees on.
    function withdrawalNetwork() public view returns (Types.WithdrawalNetwork withdrawalNetwork_) {
        (,, withdrawalNetwork_) = config();
    }

    /// @notice Network which the recipient will receive fees on.
    ///         Use the `withdrawalNetwork()` getter as this is deprecated
    ///         and is subject to be removed in the future.
    /// @custom:legacy
    function WITHDRAWAL_NETWORK() external view returns (Types.WithdrawalNetwork network_) {
        network_ = withdrawalNetwork();
    }

    /// @notice Triggers a withdrawal of funds to the fee wallet on L1 or L2.
    function withdraw() external {
        (address withdrawalRecipient, uint256 withdrawalAmount, Types.WithdrawalNetwork network) = config();

        require(
            address(this).balance >= withdrawalAmount,
            "FeeVault: withdrawal amount must be greater than minimum withdrawal amount"
        );

        uint256 value = address(this).balance;
        totalProcessed += value;

        emit Withdrawal(value, withdrawalRecipient, msg.sender);
        emit Withdrawal(value, withdrawalRecipient, msg.sender, network);

        if (network == Types.WithdrawalNetwork.L2) {
            bool success = SafeCall.send(withdrawalRecipient, value);
            require(success, "FeeVault: failed to send ETH to L2 fee recipient");
        } else {
            IL2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER)).initiateWithdrawal{ value: value }({
                _target: withdrawalRecipient,
                _gasLimit: WITHDRAWAL_MIN_GAS,
                _data: hex""
            });
        }
    }
}
