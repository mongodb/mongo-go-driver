export SERVERLESS="serverless"
export MONGODB_URI="mongodb+srv://11345-d1882aws.2tzoa.mongodb-dev.net"
export SINGLE_MONGOS_LB_URI="mongodb://invalid"
export MULTI_MONGOS_LB_URI="mongodb+srv://11345-d1882aws.2tzoa.mongodb-dev.net" 

go test ./mongo/integration -run TestCrudSpec -v | tee -a test.suite
go test ./mongo/integration -run TestRetryableWritesSpec -v | tee -a test.suite
go test ./mongo/integration -run TestUnifiedSpecs/retryable-reads -v | tee -a test.suite
go test ./mongo/integration -run TestUnifiedSpecs/sessions -v | tee -a test.suite
go test ./mongo/integration/unified -run TestUnifiedSpec/crud/unified -v | tee -a test.suite
go test ./mongo/integration/unified -run TestUnifiedSpec/transactions/unified -v | tee -a test.suite
go test ./mongo/integration/unified -run TestUnifiedSpec/versioned-api -v | tee -a test.suite
go test ./mongo/integration -run TestUnifiedSpecs/transactions/legacy -v | tee -a test.suite
go test ./mongo/integration -run TestWriteErrorsWithLabels -v | tee -a test.suite
go test ./mongo/integration -run TestWriteErrorsDetails -v | tee -a test.suite
go test ./mongo/integration -run TestHintErrors -v | tee -a test.suite
go test ./mongo/integration -run TestAggregatePrimaryPreferredReadPreference -v | tee -a test.suite
go test ./mongo/integration -run TestWriteConcernError -v | tee -a test.suite
go test ./mongo/integration -run TestErrorsCodeNamePropagated -v | tee -a test.suite
go test ./mongo/integration -run TestRetryableWritesProse -v | tee -a test.suite
go test ./mongo/integration -run TestCursor -v | tee -a test.suite