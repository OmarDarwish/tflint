package detector

import (
	"fmt"

	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/wata727/tflint/issue"
)

type AwsSecurityGroupDuplicateDetector struct {
	*Detector
	IssueType     string
	Target        string
	DeepCheck     bool
	securiyGroups map[string]bool
	defaultVpc    string
}

func (d *Detector) CreateAwsSecurityGroupDuplicateDetector() *AwsSecurityGroupDuplicateDetector {
	return &AwsSecurityGroupDuplicateDetector{
		Detector:      d,
		IssueType:     issue.ERROR,
		Target:        "aws_security_group",
		DeepCheck:     true,
		securiyGroups: map[string]bool{},
		defaultVpc:    "",
	}
}

func (d *AwsSecurityGroupDuplicateDetector) PreProcess() {
	securityGroupsResp, err := d.AwsClient.DescribeSecurityGroups()
	if err != nil {
		d.Logger.Error(err)
		d.Error = true
		return
	}
	vpcsResp, err := d.AwsClient.DescribeVpcs()
	if err != nil {
		d.Logger.Error(err)
		d.Error = true
		return
	}

	for _, securityGroup := range securityGroupsResp.SecurityGroups {
		d.securiyGroups[*securityGroup.VpcId+"."+*securityGroup.GroupName] = true
	}
	for _, vpcResource := range vpcsResp.Vpcs {
		if *vpcResource.IsDefault {
			d.defaultVpc = *vpcResource.VpcId
			break
		}
	}
}

func (d *AwsSecurityGroupDuplicateDetector) Detect(file string, item *ast.ObjectItem, issues *[]*issue.Issue) {
	nameToken, err := hclLiteralToken(item, "name")
	if err != nil {
		d.Logger.Error(err)
		return
	}
	name, err := d.evalToString(nameToken.Text)
	if err != nil {
		d.Logger.Error(err)
		return
	}
	var vpc string
	vpc, err = d.fetchVpcId(item)
	if err != nil {
		d.Logger.Error(err)
		return
	}

	if d.securiyGroups[vpc+"."+name] && !d.State.Exists(d.Target, hclObjectKeyText(item)) {
		issue := &issue.Issue{
			Type:    d.IssueType,
			Message: fmt.Sprintf("\"%s\" is duplicate name. It must be unique.", name),
			Line:    nameToken.Pos.Line,
			File:    file,
		}
		*issues = append(*issues, issue)
	}
}

func (d *AwsSecurityGroupDuplicateDetector) fetchVpcId(item *ast.ObjectItem) (string, error) {
	var vpc string
	vpcToken, err := hclLiteralToken(item, "vpc_id")
	if err != nil {
		d.Logger.Error(err)
		// "vpc_id" is optional. If omitted, use default vpc_id.
		vpc = d.defaultVpc
	} else {
		vpc, err = d.evalToString(vpcToken.Text)
		if err != nil {
			return "", err
		}
	}

	return vpc, nil
}
