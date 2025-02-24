package aliddns

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
)

// DNSManager 结构体用于管理 DNS 操作
type DNSManager struct {
	client    *alidns.Client
	region    string
	accessKey string
	secretKey string
}

// NewDNSManager 创建一个新的 DNSManager 实例
func NewDNSManager(region, accessKey, secretKey string) (*DNSManager, error) {
	client, err := alidns.NewClientWithAccessKey(region, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS client: %w", err)
	}
	return &DNSManager{
		client:    client,
		region:    region,
		accessKey: accessKey,
		secretKey: secretKey,
	}, nil
}

// ManageSubDomain 管理子域名解析记录
func (dm *DNSManager) ManageSubDomain(domainName, subDomain, recordType, recordValue string) error {
	// 检查子域名是否存在
	recordID, currentRecordValue, err := dm.checkSubDomain(domainName, subDomain, recordType)
	if err != nil {
		return fmt.Errorf("failed to check subdomain: %w", err)
	}

	// 根据检查结果处理
	if recordID == "" {
		// 子域名不存在，新建解析记录
		err = dm.addDomainRecord(domainName, subDomain, recordType, recordValue)
		if err != nil {
			return fmt.Errorf("failed to add domain record: %w", err)
		}
		fmt.Println("Domain record added successfully.")
	} else {
		// 子域名存在，检查内容是否需要更新
		if currentRecordValue != recordValue {
			// 更新解析记录
			err = dm.updateDomainRecord(recordID, subDomain, recordType, recordValue)
			if err != nil {
				return fmt.Errorf("failed to update domain record: %w", err)
			}
			fmt.Println("Domain record updated successfully.")
		} else {
			fmt.Println("No changes needed. Record is up-to-date.")
		}
	}
	return nil
}

// checkSubDomain 检查子域名是否存在
func (dm *DNSManager) checkSubDomain(domainName, subDomain, recordType string) (string, string, error) {
	request := alidns.CreateDescribeSubDomainRecordsRequest()
	request.SubDomain = fmt.Sprintf("%s.%s", subDomain, domainName)
	request.Type = recordType

	response, err := dm.client.DescribeSubDomainRecords(request)
	if err != nil {
		return "", "", err
	}

	if len(response.DomainRecords.Record) == 0 {
		// 子域名不存在
		return "", "", nil
	}

	// 子域名存在，返回记录 ID 和当前值
	record := response.DomainRecords.Record[0]
	return record.RecordId, record.Value, nil
}

// addDomainRecord 新建解析记录
func (dm *DNSManager) addDomainRecord(domainName, subDomain, recordType, recordValue string) error {
	request := alidns.CreateAddDomainRecordRequest()
	request.DomainName = domainName
	request.RR = subDomain
	request.Type = recordType
	request.Value = recordValue

	_, err := dm.client.AddDomainRecord(request)
	return err
}

// updateDomainRecord 更新解析记录
func (dm *DNSManager) updateDomainRecord(recordID, subDomain, recordType, recordValue string) error {
	request := alidns.CreateUpdateDomainRecordRequest()
	request.RecordId = recordID
	request.RR = subDomain
	request.Type = recordType
	request.Value = recordValue

	_, err := dm.client.UpdateDomainRecord(request)
	return err
}
