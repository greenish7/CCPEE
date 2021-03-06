/*
Copyright IBM Corp 2016 All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
		 http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

var pointIndexStr = "_pointindex"   //name for the key/value that will store a list of all known points
var transactionStr = "_completedtx" //name for the key/value that will store all completed transactions

var testStr = "_testIndex"

type Point struct {
	Transfer  string `json:"transfer_id"` // transfer_points_id   //the fieldtags are needed to keep case from bouncing around
	Owner     string `json:"owner"`       // User ID of owner
	Amount    string `json:"amount"`      // Amount of transfered points
	Seller    string `json:"seller"`      // Seller ID of points
	Timestamp string `json:"tr_time"`     //utc timestamp of creation
}

type Transaction struct {
	Id                  string `json:"ORD_ID"`       //Transaction ID from cppe system
	Timestamp           string `json:"EX_TIME"`      //utc timestamp of creation
	TraderA             string `json:"USER_A_ID"`    //UserA ID
	TraderB             string `json:"USER_B_ID"`    //UserB ID
	Seller              string `json:"SELLER_ID"`    //UserA's Seller ID
	Point_Amount        string `json:"POINT_AMOUNT"` //Points owned by UserA after exchange
	Prev_Transaction_id string `json:"PREV_TR_ID"`
	Expiration_Date     string `json:"EXP_DATE"`
}

type AllTx struct {
	Tid []int         `json:"tid"`
	TXs []Transaction `json:"tx"`
}
type Transac struct {
	Bid               string `json:"bid"`
	Fun               string `json:"fun"`
	Id                string `json:"id"`
	Timestamp         string `json:"timestamp"`
	TraderA           string `json:"traderA"`
	TraderB           string `json:"traderB"`
	Seller            string `json:"seller"`
	PointAmount       string `json:"pointAmount"`
	PrevTransactionID string `json:"prevTransactionId"`
	ExpDate           string `json:"expDate"`
}

type AllTxs struct {
	TXs []Transac `json:"tx"`
}
type Transact struct {
	Cert        string `json:"cert"`
	ChaincodeID string `json:"chaincodeID"`
	Nonce       string `json:"nonce"`
	Payload     string `json:"payload"`
	Signature   string `json:"signature"`
	Timestamp   string `json:"nanos"`
	Txid        string `json:"txid"`
	Type        int    `json:"type"`
}
type chart struct {
	TDs []AllTx `json:"td"`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init resets all the things
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var Aval int
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	// Initialize the chaincode
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Expecting integer value for asset holding")
	}

	// Write the state to the ledger
	err = stub.PutState("abc", []byte(strconv.Itoa(Aval))) //making a test var "abc"
	if err != nil {
		return nil, err
	}

	var empty []string
	jsonAsBytes, _ := json.Marshal(empty) //marshal an emtpy array of strings to clear the index
	err = stub.PutState(pointIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	err = stub.PutState(testStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	var trades AllTx
	jsonAsBytes, _ = json.Marshal(trades) //clear the open trade struct
	err = stub.PutState(transactionStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ============================================================================================================================
// Run - Our entry point for Invocations - [LEGACY] obc-peer 4/25/2016
// ============================================================================================================================
func (t *SimpleChaincode) Run(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("run is running " + function)
	return t.Invoke(stub, function, args)
}

// Invoke is our entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" { //initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "write" {
		return t.write(stub, args)
	} else if function == "init_transaction" {
		return t.init_transaction(stub, args)
	} else if function == "init_point" { //create a new marble
		return t.init_point(stub, args)
	} else if function == "test" {
		return t.test(stub, args)
	}
	fmt.Println("invoke did not find func: " + function) //error

	return nil, errors.New("Received unknown function invocation: " + function)
}

// Query is our entry point for queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions

	if function == "read" { //read a variable
		return t.read(stub, args)
	}

	fmt.Println("query did not find func: " + function) //error

	return nil, errors.New("Received unknown function query: " + function)
}

// ============================================================================================================================
// Read - read a variable from chaincode state
// ============================================================================================================================

func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var fun, jsonResp string
	var err error

	fun = args[0]
	if fun == "read" {
		var prid string
		q := 0
		lc := 0
		if len(args) != 2 {
			return nil, errors.New("Incorrect number of arguments. Expecting function name and name of the var to query")
		}

		//get the var from chaincode state
		if err != nil {
			jsonResp = "{\"Error\":\"Failed to get state for " + args[1] + "\"}"
			return nil, errors.New(jsonResp)
		}
		txAsbytes, err := stub.GetState(transactionStr)
		if err != nil {
			jsonResp = "{\"Error\":\"Failed to get state for " + args[1] + "\"}"
			return nil, errors.New(jsonResp)
		}

		var trans AllTx
		// Read that structure for Transaction Index
		json.Unmarshal(txAsbytes, &trans)
		rn := len(trans.TXs)
		var founded AllTx
		var foun AllTx

		for q < rn-1 {
			to := trans.TXs[q].Id
			td := trans.TXs[q+1].Id
			if to == td {
				foun.TXs = append(foun.TXs, trans.TXs[q])

				foun.TXs = append(foun.TXs, trans.TXs[q+1])
			}
			q++
		}
		vn := len(foun.TXs)
		cin := vn
		var getAll func(string, int, AllTx) AllTx

		getPrev := func(str string, tid string) (string, int, string) {
			var m, tii string
			var ind, n int
			m = "false"
			tii = ""
			n = -1
			resp, err := http.Get("https://99adba5ea39145b588cd55472f399516-vp0.us.blockchain.ibm.com:5003/transactions/" + str)
			if err != nil {
				// handle error
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			byteArray := []byte(body)
			var t Transact
			json.Unmarshal(byteArray, &t)
			st, err := base64.StdEncoding.DecodeString(t.Payload)
			if err != nil {
				log.Fatal(err)
			}
			trd := string(st)
			sp1 := strings.Replace(trd, "\n", " ", -1)
			sp := strings.Split(sp1, "\x20")
			rpl := strings.NewReplacer("$", "",
				`%`, "")
			if len(trd) > 0 {
				prid = rpl.Replace(sp[8])
				tn := sp[3]
				if tid != "" {
					//tn = tid

				}
				t := 0
				for i := 0; i < rn; i++ {
					t++
					if t > rn {
						break
					}
					a := []byte(tn)
					if len(a) > 0 {
						copy(a[0:], a[1:])
						a[len(a)-1] = 0
						a = a[:len(a)-1]
						if string(a) == trans.TXs[i].Id {
							ind = i
							break
						}

					}

				}
				return prid, ind, tn
			}
			return m, n, tii

		}

		inField := func(ssd string, spd string, trans AllTx) int {
			var ti int

			z := rn - 1
			for z >= 0 {

				a := []byte(ssd)
				if len(a) > 0 {
					copy(a[0:], a[1:])
					a[len(a)-1] = 0
					a = a[:len(a)-1]
					if string(a) == trans.TXs[z].Id && spd == trans.TXs[z].Prev_Transaction_id {
						ti = z
						return ti
					}

				}

				z--
			}
			return -1
		}

		var jsonFinal chart
		var jsonAsTrs AllTx
		//var prt AllTx
		var tid, tii, std string
		var getBranch func(string, AllTx, int)
		str := args[1]

		var n int
		co := 0
		cco := 0
		count := ""
		count2 := ""
		tidc := 0
		getAll = func(str string, ff int, prt AllTx) AllTx {
			var at Transaction
			var tk int
			tii = tid

			q = ff
			if q > -1 && str != "1" {
				tn := "0"
				td := "n0"
				to := "0n"
				if q > 0 {
					td = trans.TXs[q-1].Id
					to = trans.TXs[q].Id
					if vn > 1 && q < rn-1 {
						fmt.Println(q)
						//tn = trans.TXs[q+1].Id
					}
				}

				if to == td {
					fmt.Println("Loop 1")
					if co == 0 {
						count = trans.TXs[q-1].Prev_Transaction_id
						fmt.Println(count)
						cco++
						if tidc < vn-1 {
							at := trans.TXs[q]
							prt.TXs = append(prt.TXs, at)
							prt.Tid = append(prt.Tid, tidc)
							jsonFinal.TDs = append(jsonFinal.TDs, prt)
						}

					}
					count2 = str
					co++

					q--
					getBranch(str, prt, q)
				} else if to == tn {
					fmt.Println("Loop 2")
					if co == 0 {
						count = trans.TXs[q+1].Prev_Transaction_id
						fmt.Println(count)
						cco++
					}
					count2 = str
					co++
					q++
					getBranch(str, prt, q)
				} else {
					fmt.Println("Loop 3" + tid)
					if ff != 0 {
						str, q, tii = getPrev(str, "")
						tk = inField(tii, str, trans)
						at = trans.TXs[q]

						prt.TXs = append(prt.TXs, at)
						prt.Tid = append(prt.Tid, tidc)
						tidc++
						jsonFinal.TDs = append(jsonFinal.TDs, prt)
						jsonAsTrs = getAll(str, tk, founded)
					} else {
						at = trans.TXs[rn-1]
						prt.TXs = append(prt.TXs, at)
						prt.Tid = append(prt.Tid, tidc)
						tidc++
						jsonFinal.TDs = append(jsonFinal.TDs, prt)
						str, q, tii = getPrev(str, "")
						jsonAsTrs = getAll(str, q, founded)
					}
				}

			}

			if str == "1" && co > 0 && lc == 0 {
				fmt.Println("Loop 4")
				at = trans.TXs[ff]
				prt.TXs = append(prt.TXs, at)
				prt.Tid = append(prt.Tid, tidc)
				tidc++
				lc++
			}

			if co > 0 {
				fmt.Println("Loop 5")
				co--
				if cco == 0 {
					count = count2
					tidc = 10
					fmt.Println(count)
				}
				cco--
				str, q, tii = getPrev(count, "")
				tk = inField(tii, str, trans)
				at = trans.TXs[tk]

				prt.TXs = append(prt.TXs, at)
				prt.Tid = append(prt.Tid, tidc)
				tidc++
				jsonFinal.TDs = append(jsonFinal.TDs, prt)

				jsonAsTrs = getAll(str, tk, founded)

			}
			return prt

		}
		hk := 0
		getBranch = func(str string, jsonAsTr AllTx, q int) {
			tn := "0"
			td := trans.TXs[q-1].Id
			to := trans.TXs[q].Id
			if vn > 1 {
				tn = trans.TXs[q+1].Id
			}
			if to == td {
				fmt.Println("Branch 3")
				hk++
				if hk < vn {
					q--
					jsonAsTrs = getAll(trans.TXs[q].Prev_Transaction_id, q, founded)
					cin--
				}
			} else if to == tn {
				fmt.Println("Branch 4")
				hk++
				if hk < vn {
					q++
					jsonAsTrs = getAll(trans.TXs[q].Prev_Transaction_id, q, founded)
					cin--
				}

			}
			tidc = 10
			return
		}
		std, n, tid = getPrev(str, "")
		fmt.Println(n)
		fmt.Println(std)
		fmt.Println(tid)

		if n == rn-1 {
			//n = inField("", std, trans)
			fmt.Println(n)
			fmt.Println(std)
			fmt.Println(tid)
			jsonAsTrs = getAll(str, n, founded)
		} else {
			fmt.Println(n)
			fmt.Println(std)
			fmt.Println(tid)
			jsonAsTrs = getAll(str, n+1, founded)
		}
		jsonAsBy, _ := json.Marshal(jsonFinal)
		return jsonAsBy, nil
	} else if fun == "findLatestBySeller" {
		if len(args) != 3 {
			return nil, errors.New("Incorrect number of arguments. Expecting function name and name of the var to query")
		}

		// Numeric conversions are Atoi (string to int
		seller, err := strconv.Atoi(args[1])
		//seller = args[1]
		limit, err := strconv.Atoi(args[2])

		// Check Transaction index if it's not empty
		txAsbytes, err := stub.GetState(transactionStr)
		if err != nil {
			jsonResp = "{\"Error\":\"Failed to get state for " + args[1] + "\"}"
			return nil, errors.New(jsonResp)
		}

		// Start logic
		// Create a var from Transaction structure
		var trans AllTx
		// Read that structure for Transaction Index
		json.Unmarshal(txAsbytes, &trans)

		var founded AllTx

		for i := range trans.TXs {
			// Amashia problema
			seller_cc, err := strconv.Atoi(trans.TXs[i].Seller)
			//seller_cc_B,err := strconv.Atoi(trans.TXs[i].SellerB)
			//seller_cc_A = trans.TXs[i].SellerA
			//seller_cc_B = trans.TXs[i].SellerB
			if err == nil {
			}
			if seller_cc == seller {
				founded.TXs = append(founded.TXs, trans.TXs[i])
			}
		}
		var fulLen = len(founded.TXs)
		if limit < fulLen {
			founded.TXs = founded.TXs[fulLen-limit:]
			jsonAsBytes, _ := json.Marshal(founded)

			return jsonAsBytes, nil
		} else {
			jsonAsBytes, _ := json.Marshal(founded)
			return jsonAsBytes, nil
		}
	} /* else if fun=="findLatestTrs"{
		if len(args) != 3 {
			return nil, errors.New("Incorrect number of arguments. Expecting function name and name of the var to query")
		}
	}*/
	return nil, err //send it onward
}

// ============================================================================================================================
// Write - write variable into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) write(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var name, value string // Entities
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the variable and value to set")
	}

	name = args[0] //rename for funsies
	value = args[1]
	err = stub.PutState(name, []byte(value)) //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// ============================================================================================================================
// Init Point - create a new record of points ownership, who owns what? store into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) init_point(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//   0        		1
	// "SellerXhash", "Owner"
	if len(args) != 5 {
		return nil, errors.New("Incorrect number of arguments. Expecting 5")
	}

	//input sanitation
	fmt.Println("- start init point")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}

	if len(args[2]) <= 0 {
		return nil, errors.New("3rd argument must be a non-empty string")
	}

	if len(args[3]) <= 0 {
		return nil, errors.New("4th argument must be a non-empty string")
	}

	if len(args[4]) <= 0 {
		return nil, errors.New("5th argument must be a non-empty string")
	}

	transfer_id := args[0]
	//owner := strings.ToLower(args[1])
	owner := args[1]
	amount := args[2]
	seller := args[3]
	tr_time := args[4]

	//check if points record already exists
	pointAsBytes, err := stub.GetState(transfer_id)
	if err != nil {
		return nil, errors.New("Failed to get point id")
	}

	res := Point{}
	json.Unmarshal(pointAsBytes, &res)
	if res.Transfer == transfer_id {
		fmt.Println("This point arleady exists: " + transfer_id)
		fmt.Println(res)
		return nil, errors.New("This point arleady exists") //all stop a marble by this name exists
	}

	//build the point json string manually
	str := `{"transfer_id": "` + transfer_id + `", "owner": "` + owner + `", "amount": "` + amount + `", "seller": "` + seller + `", "timestamp": "` + tr_time + `"}`
	//str := `{"transfer_id": "` + transfer_id + `", "owner": "` + owner + `", "amount": "` + amount + `}`
	//str := `{"transfer_id": "` + transfer_id + `", "owner": "` + owner + `"}`
	err = stub.PutState(transfer_id, []byte(str)) //store Points with id as key
	if err != nil {
		return nil, err
	}

	//get the points index
	pointAsByte, err := stub.GetState(pointIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get point index")
	}
	var pointIndex []string
	json.Unmarshal(pointAsByte, &pointIndex) //un stringify it aka JSON.parse()

	//append
	pointIndex = append(pointIndex, transfer_id) //add points name to index list
	fmt.Println("! point index: ", pointIndex)
	jsonAsBytes, _ := json.Marshal(pointIndex)
	err = stub.PutState(pointIndexStr, jsonAsBytes) //store name of Points (id of transfer)

	fmt.Println("- end init marble")
	return nil, nil
}

// ============================================================================================================================
// Init Transaction - create a new record of transaction, who send to who? and what? store into chaincode state
// ============================================================================================================================

func (t *SimpleChaincode) init_transaction(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error
	//	0        1      2     3      4      5       6
	//["000009", "claudio", "alex", "Taobao", "TMall", "500", "600", "4453", "4456", "22/3/2017 2:34:12"]

	completed := Transaction{}
	completed.Id = args[0]
	completed.TraderA = args[1]
	completed.TraderB = args[2]
	completed.Seller = args[3]
	completed.Point_Amount = args[4]
	completed.Prev_Transaction_id = args[5]
	completed.Timestamp = args[6]
	completed.Expiration_Date = args[7]

	fmt.Println("- start completed trade")
	jsonAsBytes, _ := json.Marshal(completed)
	err = stub.PutState("_debug1", jsonAsBytes) // Write completed transaction under the key _debug1

	//get the completed trade struct
	tradesAsBytes, err := stub.GetState(transactionStr)
	if err != nil {
		return nil, errors.New("Failed to get TXs")
	}

	var trades AllTx
	json.Unmarshal(tradesAsBytes, &trades) //un stringify it aka JSON.parse()

	trades.TXs = append(trades.TXs, completed) //append to completed trades
	fmt.Println("! appended completed to trades")
	jsonAsBytes, _ = json.Marshal(trades)
	err = stub.PutState(transactionStr, jsonAsBytes) //rewrite completed orders
	if err != nil {
		return nil, err
	}
	fmt.Println("- end completed trade ")
	return nil, nil
}

func (t *SimpleChaincode) test(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//   0       1
	// "name", "bob"
	if len(args) < 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2")
	}

	fmt.Println("- start test fcn")
	fmt.Println(args[0] + " - " + args[1])

	//get the open trade struct
	testAsBytes, err := stub.GetState(testStr)
	if err != nil {
		return nil, errors.New("Failed to get TXs")
	}
	var test []string

	json.Unmarshal(testAsBytes, &test)

	return nil, nil
}
