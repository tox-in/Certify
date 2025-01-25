package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type Enterprise struct {
	DocType           string    `json:"docType"`
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Details           string    `json:"details"`
	State             string    `json:"state"`
	CertificateID     string    `json:"certificateId"`
	CertificationDate time.Time `json:"certificationDate"`
	CertifiedBy       string    `json:"certifiedBy"`
	RevocationDate    time.Time `json:"revocationDate"`
	RevocationReason  string    `json:"revocationReason"`
	BlacklistDate     time.Time `json:"blacklistDate"`
	BlacklistReason   string    `json:"blacklistReason"`
	Organizations     []string  `json:"organizations"`
	Channels          []string  `json:"channels"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	fmt.Println("Ledger Initialization")
	return nil
}

func (s *SmartContract) RegisterEnterprise(ctx contractapi.TransactionContextInterface, id string, name string, details string) error {
	err := checkRole(ctx, "registrar")
	if err != nil {
		return err
	}

	exists, err := s.EnterpriseExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the enterprise %s already exists", id)
	}

	enterprise := Enterprise{
		DocType:       "enterprise",
		ID:            id,
		Name:          name,
		Details:       details,
		State:         "REGISTERED",
		CertificateID: "",
		Organizations: make([]string, 0),
		Channels:      make([]string, 0),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	enterpriseJSON, err := json.Marshal(enterprise)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, enterpriseJSON)
}

func (s *SmartContract) EnterpriseExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	enterpriseJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return enterpriseJSON != nil, nil
}

func (s *SmartContract) CertifyEnterprise(ctx contractapi.TransactionContextInterface, id string) error {
	err := checkRole(ctx, "certifier")
	if err != nil {
		return err
	}

	enterprise, err := s.QueryEnterprise(ctx, id)
	if err != nil {
		return err
	}

	if enterprise.State != "REGISTERED" {
		return fmt.Errorf("enterprise %s is not in REGISTERED state", id)
	}

	enterprise.State = "CERTIFIED"
	enterprise.CertificationDate = time.Now()
	enterprise.CertificateID = generateCertificateID()
	enterprise.UpdatedAt = time.Now()

	enterpriseJSON, err := json.Marshal(enterprise)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, enterpriseJSON)
}

func (s *SmartContract) RevokeCertification(ctx contractapi.TransactionContextInterface, id string, reason string) error {
	err := checkRole(ctx, "certifier")
	if err != nil {
		return err
	}

	enterprise, err := s.QueryEnterprise(ctx, id)
	if err != nil {
		return err
	}

	if enterprise.State != "CERTIFIED" {
		return fmt.Errorf("enterprise %s is not CERTIFIED", id)
	}

	enterprise.State = "REVOKED"
	enterprise.RevocationDate = time.Now().UTC()
	enterprise.RevocationReason = reason
	enterprise.UpdatedAt = time.Now().UTC()

	enterpriseJSON, err := json.Marshal(enterprise)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, enterpriseJSON)
}

func (s *SmartContract) BlacklistEnterprise(ctx contractapi.TransactionContextInterface, id string, reason string) error {
	err := checkRole(ctx, "admin")
	if err != nil {
		return err
	}

	enterprise, err := s.QueryEnterprise(ctx, id)
	if err != nil {
		return err
	}

	if enterprise.State == "BLACKLISTED" {
		return fmt.Errorf("enterprise %s is already blacklisted", id)
	}

	previousState := enterprise.State

	enterprise.State = "BLACKLISTED"
	enterprise.BlacklistDate = time.Now().UTC()
	enterprise.BlacklistReason = reason
	enterprise.UpdatedAt = time.Now().UTC()

	enterprise.Details = fmt.Sprintf("%s|PREVIOUS_STATE:%s", enterprise.Details, previousState)

	enterpriseJSON, err := json.Marshal(enterprise)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, enterpriseJSON)
}

func (s *SmartContract) UnblacklistEnterprise(ctx contractapi.TransactionContextInterface, id string) error {
	err := checkRole(ctx, "admin")
	if err != nil {
		return err
	}

	enterprise, err := s.QueryEnterprise(ctx, id)
	if err != nil {
		return err
	}

	if enterprise.State != "BLACKLISTED" {
		return fmt.Errorf("enterprise %s is failing to bee blacklisted", id)
	}

	detailsParts := strings.Split(enterprise.Details, "|PREVIOUS_STATE:")
	if len(detailsParts) != 2 {
		return fmt.Errorf("unable to determine previous state for enterprise %s", id)
	}

	enterprise.State = detailsParts[1]
	enterprise.Details = detailsParts[0]
	enterprise.BlacklistDate = time.Time{}
	enterprise.BlacklistReason = ""
	enterprise.UpdatedAt = time.Now().UTC()

	enterpriseJSON, err := json.Marshal(enterprise)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, enterpriseJSON)
}

func (s *SmartContract) AssignOrganizations(ctx contractapi.TransactionContextInterface, id string, organizations []string) error {
	err := checkRole(ctx, "admin")
	if err != nil {
		return err
	}

	enterprise, err := s.QueryEnterprise(ctx, id)
	if err != nil {
		return err
	}

	enterprise.Organizations = organizations
	enterprise.UpdatedAt = time.Now()

	enterpriseJSON, err := json.Marshal(enterprise)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, enterpriseJSON)
}

func (s *SmartContract) AssignChannels(ctx contractapi.TransactionContextInterface, id string, channels []string) error {
	err := checkRole(ctx, "admin")
	if err != nil {
		return err
	}

	enterprise, err := s.QueryEnterprise(ctx, id)
	if err != nil {
		return err
	}

	enterprise.Channels = channels
	enterprise.UpdatedAt = time.Now()

	enterpriseJSON, err := json.Marshal(enterprise)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, enterpriseJSON)
}

func (s *SmartContract) QueryEnterprise(ctx contractapi.TransactionContextInterface, id string) (*Enterprise, error) {
	enterpriseJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if enterpriseJSON == nil {
		return nil, fmt.Errorf("the enterprise %s does not exist", id)
	}

	var enterprise Enterprise
	err = json.Unmarshal(enterpriseJSON, &enterprise)
	if err != nil {
		return nil, err
	}

	return &enterprise, nil
}

func (s *SmartContract) QueryBlacklistedEnterprises(ctx contractapi.TransactionContextInterface) ([]*Enterprise, error) {
	queryString := fmt.Sprintf(`{"selector":{"docType":"enterprise", "state":"BLACKLISTED"}}`)
	return getQueryResultForQueryString(ctx, queryString)
}

func getQueryResultForQueryString(ctx contractapi.TransactionContextInterface, queryString string) ([]*Enterprise, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var enterprises []*Enterprise
	for resultsIterator.HasNext() {
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var enterprise Enterprise
		err = json.Unmarshal(queryResult.Value, &enterprise)
		if err != nil {
			return nil, err
		}

		enterprises = append(enterprises, &enterprise)
	}
	return enterprises, nil
}
func checkRole(ctx contractapi.TransactionContextInterface, requiredRole string) error {
	clientID, err := cid.GetID(ctx.GetStub())
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	role, ok, err := cid.GetAttributeValue(ctx.GetStub(), "role")
	if err != nil {
		return fmt.Errorf("failed to get role attribute: %v", err)
	}
	if !ok {
		return fmt.Errorf("client %s does not have role attribute", clientID)
	}
	if role != requiredRole {
		return fmt.Errorf("client %s does not have required role: %s", clientID, requiredRole)
	}

	return nil
}

func generateCertificateID() string {
	return fmt.Sprintf("CERT-%d", time.Now().UnixNano())
}

func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		fmt.Printf("Error creating enterprise certification chaincode: %v", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting enterprise certification chaincode: %v", err)
	}
}
