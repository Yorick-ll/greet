package block

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"greet/model/solmodel"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/program/metaplex/token_metadata"
	"github.com/blocto/solana-go-sdk/program/token"
	"github.com/blocto/solana-go-sdk/rpc"
	"github.com/gagliardetto/solana-go"
	ag_rpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/shopspring/decimal"
)

type State struct {
	Authority       *string `json:"authority"` // 使用指针以处理可能为 null 的情况
	MetadataAddress string  `json:"metadataAddress,omitempty"`
	// AdditionalMetadata []string `json:"additionalMetadata"`
	Mint            string `json:"mint"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	UpdateAuthority string `json:"updateAuthority"`
	Uri             string `json:"uri"`
}

type Extension struct {
	Extension string `json:"extension"`
	State     State  `json:"state"`
}

type Info struct {
	Decimals        int         `json:"decimals"`
	Extensions      []Extension `json:"extensions"`
	FreezeAuthority *string     `json:"freezeAuthority"` // 使用指针以处理可能为 null 的情况
	IsInitialized   bool        `json:"isInitialized"`
	MintAuthority   string      `json:"mintAuthority"`
	Supply          string      `json:"supply"`
}

type Parsed struct {
	Info Info   `json:"info"`
	Type string `json:"type"`
}

type MintResponse struct {
	Parsed  Parsed `json:"parsed"`
	Program string `json:"program"`
	Space   int    `json:"space"`
}

type TokenUriData struct {
	Twitter     string     `json:"twitter"`
	Website     string     `json:"website"`
	Telegram    string     `json:"telegram"`
	Name        string     `json:"name"`
	Image       string     `json:"image"`
	Symbol      string     `json:"symbol"`
	Description string     `json:"description"`
	Extensions  Extensions `json:"extensions"`
}

// Extensions 结构体
type Extensions struct {
	Website  string `json:"website"`
	Twitter  string `json:"twitter"`
	Telegram string `json:"telegram"`
}

type TokenInfo struct {
	token.MintAccount
	token_metadata.Data
	MetaData      token_metadata.Metadata
	Uri           TokenUriData
	TotalSupply   decimal.Decimal
	IsCanAddToken uint8
	IsDropFreeze  uint8
	HoldersCount  int64
}

func (s *BlockService) SaveToken(ctx context.Context, trade *TradeWithPair) (tokenDB *solmodel.Token, err error) {
	tokenModel := s.sc.TokenModel
	chainId := SolChainIdInt
	tokenDB, err = tokenModel.FindOneByChainIdAddress(ctx, int64(chainId), trade.PairInfo.TokenAddr)

	solClient := s.sc.GetSolClient()

	opts := &jsonrpc.RPCClientOpts{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	rpcClient := jsonrpc.NewClientWithOpts(s.sc.Config.Sol.NodeUrl[0], opts)

	if err == nil && tokenDB != nil {

		change := false

		if tokenDB.Slot == 0 {
			tokenDB.Slot = trade.Slot
			change = true
		}

		if tokenDB.TotalSupply == 0 {
			totalSupply, err := GetTokenTotalSupply(solClient, s.ctx, tokenDB.Address)
			s.Infof("SaveToken:GetTokenMeta update token totalSupply: token addr: %v, totalSupply: %v", tokenDB.Address, totalSupply)
			if err == nil {
				tokenDB.TotalSupply = totalSupply.InexactFloat64()
				change = true
			} else {
				s.Errorf("SaveToken:GetTokenTotalSupply update err:%v, address: %v", err, tokenDB.Address)
			}
		}
		if len(tokenDB.Program) == 0 {
			program, _ := GetTokenProgram(solClient, s.ctx, tokenDB.Address)
			switch program {
			case common.TokenProgramID:
				tokenDB.Program = common.TokenProgramID.String()
				change = true
			case common.Token2022ProgramID:
				tokenDB.Program = common.Token2022ProgramID.String()
				change = true
			default:

			}
		}
		// todo: support token 2022 https://solscan.io/token/7atgF8KQo4wJrD5ATGX7t1V2zVvykPJbFfNeVf1icFv1#metadata
		if len(tokenDB.Symbol) == 0 || len(tokenDB.Name) == 0 {
			switch tokenDB.Program {
			case common.TokenProgramID.String():
				tokenInfo, err := GetTokenInfo(solClient, s.ctx, tokenDB.Address)
				if err != nil {
					s.Errorf("SaveToken:GetTokenInfo update err: %v, address: %v", err, tokenDB.Address)
				}
				if tokenInfo != nil {
					tokenDB.Symbol = tokenInfo.Data.Symbol
					tokenDB.Name = tokenInfo.Data.Name
					tokenDB.TwitterUsername = tokenInfo.Uri.Twitter
					tokenDB.Website = tokenInfo.Uri.Website
					tokenDB.Telegram = tokenInfo.Uri.Telegram
					tokenDB.Icon = tokenInfo.Uri.Image
					tokenDB.Description = tokenInfo.Uri.Description

					if len(tokenInfo.Uri.Symbol) > 0 {
						tokenDB.Symbol = tokenInfo.Uri.Symbol
					}
					if len(tokenInfo.Uri.Name) > 0 {
						tokenDB.Name = tokenInfo.Uri.Name
					}

					change = true
					s.Infof("update parse token address: %v,result: %v", tokenDB.Address, tokenDB)
				}
			case common.Token2022ProgramID.String():

				_, tokenInfo, err := GetToken2022Info(ag_rpc.NewWithCustomRPCClient(rpcClient), s.ctx, solana.MustPublicKeyFromBase58(tokenDB.Address))
				if err != nil {
					s.Errorf("SaveToken:GetToken2022Info err: %v, token address: %v", err, tokenDB.Address)
				}

				if tokenInfo != nil {

					tokenDB.Symbol = tokenInfo.Data.Symbol
					tokenDB.Name = tokenInfo.Data.Name

					tokenDB.TwitterUsername = tokenInfo.Uri.Twitter
					tokenDB.Website = tokenInfo.Uri.Website
					tokenDB.Telegram = tokenInfo.Uri.Telegram
					tokenDB.Icon = tokenInfo.Uri.Image

					tokenDB.Description = tokenInfo.Uri.Description

					if len(tokenInfo.Uri.Name) > 0 {
						tokenDB.Name = tokenInfo.Uri.Name
					}

					if len(tokenInfo.Uri.Symbol) > 0 {
						tokenDB.Symbol = tokenInfo.Uri.Symbol
					}

					change = true

					s.Infof("update parse token2022 address: %v,result: %v", tokenDB.Address, tokenDB)
				}

			default:
			}

		}

		if change {
			_ = tokenModel.Update(s.ctx, tokenDB)
		}
		// maybe zero https://solscan.io/token/nosXBVoaCTtYdLvKY6Csb4AC8JCdQKKAaWYtx2ZMoo7?program_id=675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8&sortBy=total_volumn_24h#markets

		return tokenDB, nil
	}

	if errors.Is(err, solmodel.ErrNotFound) {

		tokenDB = &solmodel.Token{
			ChainId:  int64(chainId),
			Address:  trade.PairInfo.TokenAddr,
			Decimals: int64(trade.PairInfo.TokenDecimal),
			Slot:     trade.Slot,
		}

		totalSupply, err := GetTokenTotalSupply(solClient, s.ctx, tokenDB.Address)
		if err == nil {
			tokenDB.TotalSupply = totalSupply.InexactFloat64()
		} else {
			s.Errorf("SaveToken:GetTokenTotalSupply insert err:%v, address: %v", err, tokenDB.Address)
		}

		program, _ := GetTokenProgram(solClient, s.ctx, tokenDB.Address)
		switch program {
		case common.Token2022ProgramID:
			tokenDB.Program = common.Token2022ProgramID.String()

			_, tokenInfo, err := GetToken2022Info(ag_rpc.NewWithCustomRPCClient(rpcClient), s.ctx, solana.MustPublicKeyFromBase58(tokenDB.Address))
			if err != nil {
				s.Errorf("SaveToken:GetToken2022Info err: %v, token address: %v", err, tokenDB.Address)
			}

			if tokenInfo != nil {

				tokenDB.Symbol = tokenInfo.Data.Symbol
				tokenDB.Name = tokenInfo.Data.Name

				tokenDB.TwitterUsername = tokenInfo.Uri.Twitter
				tokenDB.Website = tokenInfo.Uri.Website
				tokenDB.Telegram = tokenInfo.Uri.Telegram
				tokenDB.Icon = tokenInfo.Uri.Image
				tokenDB.Description = tokenInfo.Uri.Description

				if len(tokenInfo.Uri.Name) > 0 {
					tokenDB.Name = tokenInfo.Uri.Name
				}

				if len(tokenInfo.Uri.Symbol) > 0 {
					tokenDB.Symbol = tokenInfo.Uri.Symbol
				}
			}
			s.Infof("insert parse token2022 address: %v,result: %v", tokenDB.Address, tokenDB)
		default:
			tokenDB.Program = common.TokenProgramID.String()

			// todo: error 	SaveToken:GetTokenInfo nil,insert , err: GetTokenInfo:GetAccountInfo token data is nil,
			tokenInfo, err := GetTokenInfo(solClient, s.ctx, tokenDB.Address)
			if err != nil {
				s.Errorf("SaveToken:GetTokenInfo err: %v, address: %v", err, tokenDB.Address)
			}
			if tokenInfo != nil {
				tokenDB.Symbol = tokenInfo.Data.Symbol
				tokenDB.Name = tokenInfo.Data.Name
				tokenDB.TwitterUsername = tokenInfo.Uri.Twitter
				tokenDB.Website = tokenInfo.Uri.Website
				tokenDB.Telegram = tokenInfo.Uri.Telegram
				tokenDB.Icon = tokenInfo.Uri.Image
				tokenDB.Description = tokenInfo.Uri.Description

				if len(tokenInfo.Uri.Symbol) > 0 {
					tokenDB.Symbol = tokenInfo.Uri.Symbol
				}
				if len(tokenInfo.Uri.Name) > 0 {
					tokenDB.Name = tokenInfo.Uri.Name
				}
				// tokenDB.SetSolTokenDefaultCa()
				// tokenDB.IsCanAddToken = int64(tokenInfo.IsCanAddToken)
			}
			s.Infof("insert parse token address: %v,result: %v", tokenDB.Address, tokenDB)
		}

		err = tokenModel.Insert(ctx, tokenDB)
		if err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") {
				// db already exists
				tokenDB, err = tokenModel.FindOneByChainIdAddress(ctx, int64(chainId), trade.PairInfo.TokenAddr)
				if err != nil {
					return nil, err
				}
				return tokenDB, nil
			}
			return nil, err
		}

		s.Infof("SaveToken:GetTokenInfo insert success address: %v, token: %#v, program: %v", tokenDB.Address, tokenDB, program)

		return tokenDB, nil
	}

	return nil, fmt.Errorf("SaveToken:FindOneByChainIdAddressSymbolForUpdate err:%w", err)
}

func GetToken2022Info(c *ag_rpc.Client, ctx context.Context, address solana.PublicKey) (token2022Info *Info, tokenInfo *TokenInfo, err error) {
	resp, err := c.GetAccountInfoWithOpts(ctx, address, &ag_rpc.GetAccountInfoOpts{
		Encoding:   solana.EncodingJSONParsed,
		Commitment: ag_rpc.CommitmentConfirmed,
	})
	if err != nil || resp == nil {
		err = fmt.Errorf("GetToken2022Info:GetAccountInfoWithOpts token err:%v, token address: %v", err, address)
		return
	}

	if len(resp.Value.Data.GetRawJSON()) == 0 {
		err = fmt.Errorf("GetToken2022Info:GetAccountInfoWithOpts token data is nil, err:%v, token address: %#v", err, address)
		return nil, nil, err
	}

	mintResponse, err := Byte2Struct[*MintResponse](resp.Value.Data.GetRawJSON())
	if err != nil {
		err = fmt.Errorf("GetToken2022Info:Byte2Struct err:%v, token address: %v", err, address)
		return
	}

	tokenInfo = &TokenInfo{}

	for _, extension := range mintResponse.Parsed.Info.Extensions {
		if extension.Extension == "tokenMetadata" {
			tokenInfo.Name = extension.State.Name
			tokenInfo.Symbol = extension.State.Symbol

			if len(extension.State.Uri) > 0 {
				publicGateway := "https://ipfs.io/ipfs/"
				if !isURLAccessible(extension.State.Uri) {
					extension.State.Uri = replaceWithPublicGateway(extension.State.Uri, publicGateway)
				}

				ctx, cancelFunc := context.WithTimeout(context.Background(), 3000*time.Millisecond)
				defer cancelFunc()
				request, err := http.NewRequestWithContext(ctx, http.MethodGet, extension.State.Uri, nil)
				if err != nil {
					err = fmt.Errorf("http.NewRequest err:%w", err)
					return token2022Info, tokenInfo, err
				}

				// 执行请求
				response, err := http.DefaultClient.Do(request)
				if err != nil {
					// skip error
					return token2022Info, tokenInfo, nil
				}
				defer func() {
					_ = response.Body.Close()
				}()

				res, err := io.ReadAll(response.Body)

				if err != nil {
					// skip error
					return token2022Info, tokenInfo, nil
				}
				// 检查 Content-Type
				contentType := response.Header.Get("Content-Type")
				if strings.Contains(string(res), "Account has been disabled.") {
					return token2022Info, tokenInfo, nil
				}
				if strings.HasPrefix(contentType, "application/json") {
					tokenUriData, err := Byte2Struct[TokenUriData](res)
					if err != nil {
						return token2022Info, tokenInfo, nil
					}

					if len(tokenUriData.Website) == 0 {
						tokenUriData.Website = tokenUriData.Extensions.Website
					}
					if len(tokenUriData.Telegram) == 0 {
						tokenUriData.Telegram = tokenUriData.Extensions.Telegram
					}
					if len(tokenUriData.Twitter) == 0 {
						tokenUriData.Twitter = tokenUriData.Extensions.Twitter
					}

					tokenInfo.Uri = tokenUriData
				} else if strings.HasPrefix(contentType, "image/") {
					// maybe picture
					// https://solscan.io/token/2HPtzSqkivqk8P5ySqVxB17b93sXsJN4s77kJp4Eish9#metadata
					// if strings.Contains(err.Error(), "invalid character") {
					// 	tokenInfo.Uri.Image = metaData.Data.Uri
					// }
					// skip error
					tokenInfo.Uri.Image = extension.State.Uri
				} else {

					tokenUriData, err := Byte2Struct[TokenUriData](res)
					if err != nil {
						err = fmt.Errorf("GetToken2022Info error: %v, url: %v, token address: %v", err, extension.State.Uri, address)
						return token2022Info, tokenInfo, err
					}

					if len(tokenUriData.Website) == 0 {
						tokenUriData.Website = tokenUriData.Extensions.Website
					}
					if len(tokenUriData.Telegram) == 0 {
						tokenUriData.Telegram = tokenUriData.Extensions.Telegram
					}
					if len(tokenUriData.Twitter) == 0 {
						tokenUriData.Twitter = tokenUriData.Extensions.Twitter
					}

					tokenInfo.Uri = tokenUriData
				}

			}
		}
	}

	_ = mintResponse

	return &mintResponse.Parsed.Info, tokenInfo, nil
}

func GetTokenTotalSupply(c *client.Client, ctx context.Context, address string) (decimal.Decimal, error) {
	supplyModel, err := c.GetTokenSupplyWithConfig(ctx, address, client.GetTokenSupplyConfig{
		Commitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		err = fmt.Errorf("GetTokenTotalSupply token err:%v,token address: %v", err, address)
		return decimal.Zero, err
	}
	totalSupply := decimal.NewFromUint64(supplyModel.Amount).Div(decimal.New(1, int32(supplyModel.Decimals)))
	return totalSupply, nil
}

func GetTokenInfo(c *client.Client, ctx context.Context, address string) (tokenInfo *TokenInfo, err error) {
	resp, err := c.GetAccountInfoWithConfig(ctx, address, client.GetAccountInfoConfig{
		Commitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		err = fmt.Errorf("GetAccountInfoWithConfig token err:%v, token address: %v", err, address)
		return
	}

	if len(resp.Data) == 0 {
		err = fmt.Errorf("GetTokenInfo:GetAccountInfoWithConfig token data is nil, err:%v, token address: %#v", err, address)
		return nil, err
	}

	mintAccount, err := token.MintAccountFromData(resp.Data[:82])
	if err != nil {
		err = fmt.Errorf("GetTokenInfo:MintAccountFromData err:%v, token address: %v", err, address)
		return
	}

	tokenInfo = &TokenInfo{
		MintAccount: mintAccount,
	}
	// 在 solana 上，元数据账户是由 Token Metadata Program 管理的，该程序的地址是固定的：
	// Token Metadata Program ID: metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s
	// 元数据账户的地址是由代币的 Mint 地址和一组固定的种子（seeds）通过程序派生地址 (Program Derived Address, PDA) 的方式计算得出的。
	// 	1.	固定种子
	// Token Metadata Program 使用以下种子来派生元数据账户地址：
	//	•	metadata（固定字符串，表示账户类型）
	//	•	Token Metadata Program ID
	//	•	Token 的 Mint 地址
	//	2.	PDA 计算
	// 使用 Solana 的 PDA 规则，结合上述种子，计算出元数据账户地址。
	meta, err := token_metadata.GetTokenMetaPubkey(common.PublicKeyFromString(address))
	if err != nil {
		err = fmt.Errorf("GetTokenMetaPubkey err:%w", err)
		return
	}
	metaAccount, err := c.GetAccountInfoWithConfig(ctx, meta.String(), client.GetAccountInfoConfig{
		Commitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		err = fmt.Errorf("GetTokenInfo:GetAccountInfoWithConfig meta err:%w", err)
		return
	}
	if len(metaAccount.Data) <= 0 {
		return
	}
	metaData, err := token_metadata.MetadataDeserialize(metaAccount.Data)
	if err != nil {
		err = fmt.Errorf("deserialize metaAccount data err:%w", err)
		return
	}
	tokenInfo.MetaData = metaData
	tokenInfo.Data = metaData.Data
	// tokenInfo.TotalSupply = decimal.NewFromUint64(tokenInfo.Supply).Div(decimal.New(1, int32(tokenInfo.Decimals)))
	// if tokenInfo.MintAccount.MintAuthority != nil {
	// 	tokenInfo.IsCanAddToken = 1
	// }
	// if tokenInfo.MintAccount.FreezeAuthority == nil {
	// 	tokenInfo.IsDropFreeze = 1
	// }

	// totalHolders, err := GetTokenHolders(ctx, address)
	// if err == nil {
	// 	tokenInfo.HoldersCount = totalHolders
	// }
	// if err != nil {
	// 	logx.Errorf("GetTokenHolders %v, err:%v", address, err)
	// }

	if len(metaData.Data.Uri) > 0 {
		publicGateway := "https://ipfs.io/ipfs/"
		if !isURLAccessible(metaData.Data.Uri) {
			metaData.Data.Uri = replaceWithPublicGateway(metaData.Data.Uri, publicGateway)
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), 3000*time.Millisecond)
		defer cancelFunc()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, metaData.Data.Uri, nil)
		if err != nil {
			err = fmt.Errorf("http.NewRequest err:%w", err)
			return tokenInfo, err
		}

		// 执行请求
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			// skip error
			return tokenInfo, nil
		}
		defer func() {
			_ = response.Body.Close()
		}()

		res, err := io.ReadAll(response.Body)

		if err != nil {
			// skip error
			return tokenInfo, nil
		}
		// 检查 Content-Type
		contentType := response.Header.Get("Content-Type")
		if strings.Contains(string(res), "Account has been disabled.") {
			return tokenInfo, nil
		}
		if strings.HasPrefix(contentType, "application/json") {
			tokenUriData, err := Byte2Struct[TokenUriData](res)
			if err != nil {
				return tokenInfo, nil
			}

			if len(tokenUriData.Website) == 0 {
				tokenUriData.Website = tokenUriData.Extensions.Website
			}
			if len(tokenUriData.Telegram) == 0 {
				tokenUriData.Telegram = tokenUriData.Extensions.Telegram
			}
			if len(tokenUriData.Twitter) == 0 {
				tokenUriData.Twitter = tokenUriData.Extensions.Twitter
			}

			tokenInfo.Uri = tokenUriData
		} else if strings.HasPrefix(contentType, "image/") {
			// maybe picture
			// https://solscan.io/token/2HPtzSqkivqk8P5ySqVxB17b93sXsJN4s77kJp4Eish9#metadata
			// if strings.Contains(err.Error(), "invalid character") {
			// 	tokenInfo.Uri.Image = metaData.Data.Uri
			// }
			// skip error
			tokenInfo.Uri.Image = metaData.Data.Uri
		} else {
			// default
			tokenUriData, err := Byte2Struct[TokenUriData](res)
			if err != nil {
				// err = fmt.Errorf("GetTokenInfo error: %v, url: %v, token address: %v", err, metaData.Data.Uri, address)
				return tokenInfo, nil
			}

			if len(tokenUriData.Website) == 0 {
				tokenUriData.Website = tokenUriData.Extensions.Website
			}
			if len(tokenUriData.Telegram) == 0 {
				tokenUriData.Telegram = tokenUriData.Extensions.Telegram
			}
			if len(tokenUriData.Twitter) == 0 {
				tokenUriData.Twitter = tokenUriData.Extensions.Twitter
			}

			tokenInfo.Uri = tokenUriData

		}

	}

	return
}

// 检查 URL 是否可访问
func isURLAccessible(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// 替换为公共的 IPFS 网关
func replaceWithPublicGateway(ipfsURL string, publicGateway string) string {
	// 正则表达式来匹配 IPFS 网关，您可以根据需要进行扩展
	pattern := `^https?://[^/]+/ipfs/`

	re := regexp.MustCompile(pattern)
	if re.MatchString(ipfsURL) {
		// 替换为公共网关
		return re.ReplaceAllString(ipfsURL, publicGateway)
	}
	return ipfsURL // 如果没有匹配，返回原始 URL
}

// Byte2Struct 将字节数组解码为指定类型的结构体
func Byte2Struct[T any](b []byte) (T, error) {
	var t T
	err := json.Unmarshal(b, &t)
	return t, err
}

func GetTokenProgram(c *client.Client, ctx context.Context, address string) (program common.PublicKey, err error) {
	resp, err := c.GetAccountInfoWithConfig(ctx, address, client.GetAccountInfoConfig{
		Commitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		err = fmt.Errorf("GetTokenMintInfo token err:%v, token address: %v", err, address)
		return
	}

	switch resp.Owner {
	case common.Token2022ProgramID:
		return common.Token2022ProgramID, nil
	case common.TokenProgramID:
		return common.TokenProgramID, nil
	}
	return common.SystemProgramID, errors.New("not support")
}
