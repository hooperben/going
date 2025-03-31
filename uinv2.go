package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Structure to hold the input data
type InputRequest struct {
	pool_address       string
	input_token_address  string
	output_token_address string
	input_amount       *big.Float
}

// PoolReserves represents the result of getReserves call
type PoolReserves struct {
	_reserve0           *big.Int
	_reserve1           *big.Int
	_blockTimestampLast uint32
}

// Create a new struct to hold both balance and decimals
type TokenInfo struct {
	Balance  *big.Int
	Decimals uint8
	Symbol   string
}

func main() {
	fmt.Println("==================================================")
	fmt.Println("|          Uniswap V2 Price Bot Input CLI        |")
	fmt.Println("==================================================")
	fmt.Println("| Calculates the amount of an output token       |")
	fmt.Println("| for a given input token and amount             |")
	fmt.Println("|                                                |")
	fmt.Println("| You can hit enter on all input fields to       |")
	fmt.Println("| default to selling 1 ETH for USDT              |")
	fmt.Println("==================================================")

	var default_pool_address = "0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852"
	var default_input_token_address = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	var default_output_token_address = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	var default_input_amount = "1"

	// Get pool address
	pool_address := getAddressInput("Enter Uniswap pool address (0x...): ", default_pool_address)
	
	// Get token1 address
	input_token_address := getAddressInput("Enter Input Token address (0x...): ", default_input_token_address)
	
	// Get token2 address
	output_token_address := getAddressInput("Enter Output Token address (0x...): ", default_output_token_address)
	
	// Get input amount
	input_amount := getAmountInput("Enter input amount (human readable please, e.g. 1, 1.3 0.7777): ", default_input_amount)
	
	// Create the InputRequest structure
	input_request := InputRequest{
		pool_address:        pool_address,
		input_token_address:  input_token_address,
		output_token_address: output_token_address,
		input_amount:        input_amount,
	}

	fmt.Println("\nInput Request Summary:")
	fmt.Printf("Pool Address: %s\n", input_request.pool_address)
	fmt.Printf("Input Token Address: %s\n", input_request.input_token_address)
	fmt.Printf("Output Token Address: %s\n", input_request.output_token_address)
	fmt.Printf("Input Amount: %s\n", input_request.input_amount.Text('f', 6))

	fmt.Printf("------ Calling RPC for amount data ------\n")

	// TODO use env variable
	var ALCHEMY_API_KEY = "ywt4Fdhun2J3lH0hX5YPXqaXiBAusUxG"

	// Create an RPC instance
	client, err := ethclient.Dial("https://eth-mainnet.g.alchemy.com/v2/" + ALCHEMY_API_KEY)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	
	// get x and y values from pool
	reserves, err := getReserves(client, input_request.pool_address)
	if err != nil {
		fmt.Printf("Error getting pool reserves: %v\n", err)
		return
	} else {
		fmt.Println("\nPool Reserves:")
		fmt.Printf("Reserve0: %s\n", reserves._reserve0.String())
		fmt.Printf("Reserve1: %s\n", reserves._reserve1.String())
	}
	
	// input token details
	input_token_info, err := getTokenBalance(client, input_request.input_token_address, input_request.pool_address)
	if err != nil {
		fmt.Printf("Error getting input token info: %v\n", err)
		return
	} else {
		fmt.Printf("\nInput Token Balance in Pool: %s\n", input_token_info.Balance.String())
		fmt.Printf("Input Token Decimals: %d\n", input_token_info.Decimals)
		
		// Calculate human-readable balance
		divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(input_token_info.Decimals)), nil))
		raw_balance := new(big.Float).SetInt(input_token_info.Balance)
		readable_balance := new(big.Float).Quo(raw_balance, divisor)
		
		fmt.Printf("Input Token Symbol: %s\n", input_token_info.Symbol)
		fmt.Printf("Input Token Readable Balance: %.6f\n", readable_balance)
	}
	
	// output token details
	output_token_info, err := getTokenBalance(client, input_request.output_token_address, input_request.pool_address)
	if err != nil {
		fmt.Printf("Error getting output token info: %v\n", err)
		return
	} else {
		fmt.Printf("\nOutput Token Balance in Pool: %s\n", output_token_info.Balance.String())
		fmt.Printf("Output Token Decimals: %d\n", output_token_info.Decimals)
		
		// Calculate human-readable balance
		divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(output_token_info.Decimals)), nil))
		raw_balance := new(big.Float).SetInt(output_token_info.Balance)
		readable_balance := new(big.Float).Quo(raw_balance, divisor)
		
		fmt.Printf("Output Token Symbol: %s\n", output_token_info.Symbol)
		fmt.Printf("Output Token Readable Balance: %.6f\n", readable_balance)
	}
	
	// Determine token order in the pool (to know which reserve is which)
	token0, err := getToken0(client, input_request.pool_address)
	if err != nil {
		fmt.Printf("Error getting token0 address: %v\n", err)
		return
	}
	
	// Convert input amount to raw amount with decimals
	input_amount_raw := new(big.Int)

	// input_amount * 10 ** input_token.decimals()
	input_amount_decimal := new(big.Float).Mul(
		input_request.input_amount,
		new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(input_token_info.Decimals)), nil)),
	)

	// Convert to big.Int, discarding any fractional part (?)
	input_amount_decimal.Int(input_amount_raw) 
	
	fmt.Printf("\nInput Amount (raw with decimals): %s\n", input_amount_raw.String())
	
	// Calculate output amount using Uniswap formula
	var output_amount *big.Int

	// TODO fee is static
	// Fee is 0.3%, so r = 0.997
	fee := big.NewInt(997)
	feeBase := big.NewInt(1000)
	
	// Determine x and y based on token order
	var x, y *big.Int
	if strings.EqualFold(token0.Hex(), input_request.input_token_address) {
		// Input token is token0
		x = reserves._reserve0
		y = reserves._reserve1
		fmt.Println("Input token is token0 in the pool")
	} else {
		// Input token is token1
		x = reserves._reserve1
		y = reserves._reserve0
		fmt.Println("Input token is token1 in the pool")
	}
	
	// Calculate output amount using the formula: Δy = (y * r * Δx) / (x + r * Δx)
	numerator := new(big.Int).Mul(y, new(big.Int).Mul(fee, input_amount_raw))
	denominator := new(big.Int).Add(
		new(big.Int).Mul(x, feeBase),
		new(big.Int).Mul(fee, input_amount_raw),
	)
	output_amount = new(big.Int).Div(numerator, denominator)
	
	fmt.Printf("Output Amount (raw): %s\n", output_amount.String())
	
	// Convert output amount to human-readable format
	output_divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(output_token_info.Decimals)), nil))
	output_float := new(big.Float).SetInt(output_amount)
	readable_output := new(big.Float).Quo(output_float, output_divisor)
	
	fmt.Printf("Output Amount (human-readable): %.6f %s\n", readable_output, output_token_info.Symbol)
}

//////// CLI PARSING FUNCTIONS ////////

// Helper function to get and parse address input
func getAddressInput(prompt string, address_default string) string {
	for {
		fmt.Print(prompt)
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println("Using default address for variable: "+ address_default)
			return address_default;
		}
		
		if len(input) >= 2 && input[:2] == "0x" {
			return input
		} else {
			fmt.Println("Error: Address must start with 0x")
		}
	}
}

// Helper function to get and parse amount input - updated to handle decimals
func getAmountInput(prompt string, amount_default string) *big.Float {
	for {
		fmt.Print(prompt)
		var input string
		_, err := fmt.Scanln(&input)

		if err != nil {
			fmt.Println("Using default value for amount: "+ amount_default)
			input = amount_default;
		}
		
		amount, success := new(big.Float).SetString(input)
		if success {
			return amount
		} else {
			fmt.Println("Error: Invalid number. Please enter a valid decimal number")
		}
	}
}

//////// CONTRACT READS ////////

// Function to call getReserves() on a Uniswap pool
func getReserves(client *ethclient.Client, pool_addr_hex string) (*PoolReserves, error) {
	// Convert string address to common.Address
	pool_addr := common.HexToAddress(pool_addr_hex)
	
	// Define the ABI for getReserves
	const get_reserves_abi = `[{"constant":true,"inputs":[],"name":"getReserves","outputs":[{"internalType":"uint112","name":"_reserve0","type":"uint112"},{"internalType":"uint112","name":"_reserve1","type":"uint112"},{"internalType":"uint32","name":"_blockTimestampLast","type":"uint32"}],"payable":false,"stateMutability":"view","type":"function"}]`
	
	// Parse the ABI
	parsed_abi, err := abi.JSON(strings.NewReader(get_reserves_abi))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}
	
	// Pack the function call
	data, err := parsed_abi.Pack("getReserves")
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %v", err)
	}
	
	// Call the contract
	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &pool_addr,
		Data: data,
	}, nil)
	
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %v", err)
	}
	
	// Instead of using ABI unpacking, manually parse the bytes
	reserves, err := parseReservesFromBytes(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reserves: %v", err)
	}
	
	return reserves, nil
}

// Function to manually parse getReserves result bytes
func parseReservesFromBytes(data []byte) (*PoolReserves, error) {
	if len(data) < 96 { // Expected at least 3 x 32 bytes for the three return values
		return nil, fmt.Errorf("insufficient data length: %d", len(data))
	}
	
	// In Ethereum ABI encoding, each value takes up 32 bytes (even if it's smaller)
	reserve0 := new(big.Int).SetBytes(data[0:32])
	reserve1 := new(big.Int).SetBytes(data[32:64])
	
	// The timestamp is a uint32, but still takes up 32 bytes in the ABI encoding
	// We need just the last 4 bytes for uint32
	timestampBytes := data[64:96] // Take only the needed 4 bytes
	timestamp := binary.BigEndian.Uint32(timestampBytes)
	
	reserves := &PoolReserves{
		_reserve0:           reserve0,
		_reserve1:           reserve1,
		_blockTimestampLast: timestamp,
	}
	
	return reserves, nil
}

// Function to call balanceOf() and decimals() on a token contract
func getTokenBalance(client *ethclient.Client, token_addr_hex string, account_addr_hex string) (*TokenInfo, error) {
	// Convert string addresses to common.Address
	token_addr := common.HexToAddress(token_addr_hex)
	account_addr := common.HexToAddress(account_addr_hex)
	
	// Get balance
	balance, err := getTokenBalanceOf(client, token_addr, account_addr)
	if err != nil {
		return nil, err
	}
	
	// Get decimals
	decimals, err := getTokenDecimals(client, token_addr)
	if err != nil {
		return nil, err
	}
	
	// Get symbol
	symbol, err := getTokenSymbol(client, token_addr)
	if err != nil {
		symbol = "UNKNOWN" // Use placeholder if symbol can't be retrieved
	}
	
	return &TokenInfo{
		Balance:  balance,
		Decimals: decimals,
		Symbol:   symbol,
	}, nil
}

// Function to call balanceOf() on a token contract
func getTokenBalanceOf(client *ethclient.Client, token_addr common.Address, account_addr common.Address) (*big.Int, error) {
	// Define the ABI for balanceOf
	const balance_of_abi = `[{"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`
	
	// Parse the ABI
	parsed_abi, err := abi.JSON(strings.NewReader(balance_of_abi))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}
	
	// Pack the function call with account address as parameter
	data, err := parsed_abi.Pack("balanceOf", account_addr)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %v", err)
	}
	
	// Call the contract
	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &token_addr,
		Data: data,
	}, nil)
	
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %v", err)
	}
	
	// Manually parse the balance from the result
	if len(result) < 32 {
		return nil, fmt.Errorf("insufficient data for balance")
	}
	
	balance := new(big.Int).SetBytes(result[:32])
	return balance, nil
}

// Function to call decimals() on a token contract
func getTokenDecimals(client *ethclient.Client, token_addr common.Address) (uint8, error) {
	// Define the ABI for decimals
	const decimals_abi = `[{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"}]`
	
	// Parse the ABI
	parsed_abi, err := abi.JSON(strings.NewReader(decimals_abi))
	if err != nil {
		return 0, fmt.Errorf("failed to parse ABI: %v", err)
	}
	
	// Pack the function call
	data, err := parsed_abi.Pack("decimals")
	if err != nil {
		return 0, fmt.Errorf("failed to pack data: %v", err)
	}
	
	// Call the contract
	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &token_addr,
		Data: data,
	}, nil)
	
	if err != nil {
		return 0, fmt.Errorf("failed to call contract: %v", err)
	}
	
	// Manually parse the decimals from the result
	if len(result) < 32 {
		return 0, fmt.Errorf("insufficient data for decimals")
	}
	
	// Decimals is a uint8, so we need the last byte of the 32-byte slot
	return result[31], nil
}

// Function to call symbol() on a token contract
func getTokenSymbol(client *ethclient.Client, token_addr common.Address) (string, error) {
	// Define the ABI for symbol
	const symbol_abi = `[{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"}]`
	
	// Parse the ABI
	parsed_abi, err := abi.JSON(strings.NewReader(symbol_abi))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %v", err)
	}
	
	// Pack the function call
	data, err := parsed_abi.Pack("symbol")
	if err != nil {
		return "", fmt.Errorf("failed to pack data: %v", err)
	}
	
	// Call the contract
	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &token_addr,
		Data: data,
	}, nil)
	
	if err != nil {
		return "", fmt.Errorf("failed to call contract: %v", err)
	}
	
	// Parse the result using the ABI
	var symbol string
	err = parsed_abi.UnpackIntoInterface(&symbol, "symbol", result)
	if err != nil {
		// Try to decode manually if ABI unpacking fails
		if len(result) >= 96 {
			// The first 32 bytes contain the offset to the data
			// The next 32 bytes contain the length of the string
			// The next bytes contain the string data
			length := new(big.Int).SetBytes(result[32:64]).Uint64()
			if length > 0 && uint64(len(result)) >= 64+length {
				symbol = string(result[64 : 64+length])
			}
		}
		
		if symbol == "" {
			return "", fmt.Errorf("failed to unpack symbol: %v", err)
		}
	}
	
	return symbol, nil
}

// Function to get token0 address from a Uniswap V2 pool
func getToken0(client *ethclient.Client, pool_addr_hex string) (common.Address, error) {
	pool_addr := common.HexToAddress(pool_addr_hex)
	
	// Define the ABI for token0 function
	const token0_abi = `[{"constant":true,"inputs":[],"name":"token0","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"}]`
	
	parsed_abi, err := abi.JSON(strings.NewReader(token0_abi))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to parse ABI: %v", err)
	}
	
	data, err := parsed_abi.Pack("token0")
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to pack data: %v", err)
	}
	
	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &pool_addr,
		Data: data,
	}, nil)
	
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to call contract: %v", err)
	}
	
	if len(result) < 32 {
		return common.Address{}, fmt.Errorf("insufficient data for address")
	}
	
	// Extract the address from the last 20 bytes
	return common.BytesToAddress(result[12:32]), nil
}