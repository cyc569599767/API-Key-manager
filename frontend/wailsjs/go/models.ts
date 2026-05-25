export namespace model {
	
	export class APIKey {
	    id: number;
	    name: string;
	    provider: string;
	    environment: string;
	    groupName: string;
	    budgetTag: string;
	    baseUrl: string;
	    maskedKey: string;
	    status: string;
	    testEndpoint: string;
	    timeoutMs: number;
	    rateLimitRpm: number;
	    failureStrategy: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    // Go type: time
	    disabledAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new APIKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.environment = source["environment"];
	        this.groupName = source["groupName"];
	        this.budgetTag = source["budgetTag"];
	        this.baseUrl = source["baseUrl"];
	        this.maskedKey = source["maskedKey"];
	        this.status = source["status"];
	        this.testEndpoint = source["testEndpoint"];
	        this.timeoutMs = source["timeoutMs"];
	        this.rateLimitRpm = source["rateLimitRpm"];
	        this.failureStrategy = source["failureStrategy"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.disabledAt = this.convertValues(source["disabledAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TestResult {
	    id: number;
	    apiKeyId: number;
	    endpoint: string;
	    status: string;
	    httpStatus: number;
	    latencyMs: number;
	    modelCount: number;
	    errorRate: number;
	    errorMessage: string;
	    requestHeadersJson: string;
	    requestBodyJson: string;
	    responseHeadersJson: string;
	    responseBodyJson: string;
	    responseTruncated: boolean;
	    // Go type: time
	    testedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.apiKeyId = source["apiKeyId"];
	        this.endpoint = source["endpoint"];
	        this.status = source["status"];
	        this.httpStatus = source["httpStatus"];
	        this.latencyMs = source["latencyMs"];
	        this.modelCount = source["modelCount"];
	        this.errorRate = source["errorRate"];
	        this.errorMessage = source["errorMessage"];
	        this.requestHeadersJson = source["requestHeadersJson"];
	        this.requestBodyJson = source["requestBodyJson"];
	        this.responseHeadersJson = source["responseHeadersJson"];
	        this.responseBodyJson = source["responseBodyJson"];
	        this.responseTruncated = source["responseTruncated"];
	        this.testedAt = this.convertValues(source["testedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class APIKeyDetail {
	    key: APIKey;
	    latestTestResult?: TestResult;
	
	    static createFrom(source: any = {}) {
	        return new APIKeyDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = this.convertValues(source["key"], APIKey);
	        this.latestTestResult = this.convertValues(source["latestTestResult"], TestResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class APIKeyFilter {
	    search: string;
	    provider: string;
	    status: string;
	    environment: string;
	    groupName: string;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new APIKeyFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.search = source["search"];
	        this.provider = source["provider"];
	        this.status = source["status"];
	        this.environment = source["environment"];
	        this.groupName = source["groupName"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class APIKeyListResult {
	    items: APIKey[];
	    total: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new APIKeyListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], APIKey);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AuditLog {
	    id: number;
	    apiKeyId?: number;
	    apiKeyName?: string;
	    action: string;
	    summary: string;
	    metadataJson: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AuditLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.apiKeyId = source["apiKeyId"];
	        this.apiKeyName = source["apiKeyName"];
	        this.action = source["action"];
	        this.summary = source["summary"];
	        this.metadataJson = source["metadataJson"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupRestoreResult {
	    path: string;
	    canceled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupRestoreResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.canceled = source["canceled"];
	    }
	}
	export class BatchDeleteAPIKeySnapshot {
	    id: number;
	    name: string;
	    provider: string;
	    baseUrl: string;
	    maskedKey: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchDeleteAPIKeySnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.baseUrl = source["baseUrl"];
	        this.maskedKey = source["maskedKey"];
	    }
	}
	export class BatchDeleteAPIKeysInput {
	    ids: number[];
	
	    static createFrom(source: any = {}) {
	        return new BatchDeleteAPIKeysInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ids = source["ids"];
	    }
	}
	export class BatchDeleteAPIKeysResult {
	    total: number;
	    deleted: number;
	    skipped: number;
	    items: BatchDeleteAPIKeySnapshot[];
	
	    static createFrom(source: any = {}) {
	        return new BatchDeleteAPIKeysResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.deleted = source["deleted"];
	        this.skipped = source["skipped"];
	        this.items = this.convertValues(source["items"], BatchDeleteAPIKeySnapshot);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BatchExportAPIKeysInput {
	    ids: number[];
	
	    static createFrom(source: any = {}) {
	        return new BatchExportAPIKeysInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ids = source["ids"];
	    }
	}
	export class BatchExportAPIKeysResult {
	    total: number;
	    exported: number;
	    skipped: number;
	    path: string;
	    canceled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BatchExportAPIKeysResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.exported = source["exported"];
	        this.skipped = source["skipped"];
	        this.path = source["path"];
	        this.canceled = source["canceled"];
	    }
	}
	export class BatchImportAPIKeyFailure {
	    line: number;
	    name: string;
	    maskedKey: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchImportAPIKeyFailure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.name = source["name"];
	        this.maskedKey = source["maskedKey"];
	        this.reason = source["reason"];
	    }
	}
	export class BatchImportAPIKeySkipped {
	    line: number;
	    name: string;
	    maskedKey: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchImportAPIKeySkipped(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.name = source["name"];
	        this.maskedKey = source["maskedKey"];
	        this.reason = source["reason"];
	    }
	}
	export class CreateAPIKeyInput {
	    name: string;
	    provider: string;
	    environment: string;
	    groupName: string;
	    budgetTag: string;
	    baseUrl: string;
	    apiKey: string;
	    testEndpoint: string;
	    timeoutMs: number;
	    rateLimitRpm: number;
	    failureStrategy: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateAPIKeyInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.environment = source["environment"];
	        this.groupName = source["groupName"];
	        this.budgetTag = source["budgetTag"];
	        this.baseUrl = source["baseUrl"];
	        this.apiKey = source["apiKey"];
	        this.testEndpoint = source["testEndpoint"];
	        this.timeoutMs = source["timeoutMs"];
	        this.rateLimitRpm = source["rateLimitRpm"];
	        this.failureStrategy = source["failureStrategy"];
	    }
	}
	export class BatchImportAPIKeysInput {
	    rawText: string;
	    defaults: CreateAPIKeyInput;
	    format: string;
	    duplicateStrategy: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchImportAPIKeysInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rawText = source["rawText"];
	        this.defaults = this.convertValues(source["defaults"], CreateAPIKeyInput);
	        this.format = source["format"];
	        this.duplicateStrategy = source["duplicateStrategy"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BatchImportAPIKeysResult {
	    total: number;
	    created: number;
	    skipped: number;
	    failed: number;
	    failures: BatchImportAPIKeyFailure[];
	    skippedRows: BatchImportAPIKeySkipped[];
	    createdKeys: APIKey[];
	
	    static createFrom(source: any = {}) {
	        return new BatchImportAPIKeysResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.created = source["created"];
	        this.skipped = source["skipped"];
	        this.failed = source["failed"];
	        this.failures = this.convertValues(source["failures"], BatchImportAPIKeyFailure);
	        this.skippedRows = this.convertValues(source["skippedRows"], BatchImportAPIKeySkipped);
	        this.createdKeys = this.convertValues(source["createdKeys"], APIKey);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BatchTestAPIKeyItemResult {
	    id: number;
	    name: string;
	    success: boolean;
	    status: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchTestAPIKeyItemResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.success = source["success"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
	export class BatchTestAPIKeysInput {
	    ids: number[];
	
	    static createFrom(source: any = {}) {
	        return new BatchTestAPIKeysInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ids = source["ids"];
	    }
	}
	export class BatchTestAPIKeysResult {
	    total: number;
	    success: number;
	    failed: number;
	    skipped: number;
	    results: BatchTestAPIKeyItemResult[];
	
	    static createFrom(source: any = {}) {
	        return new BatchTestAPIKeysResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.success = source["success"];
	        this.failed = source["failed"];
	        this.skipped = source["skipped"];
	        this.results = this.convertValues(source["results"], BatchTestAPIKeyItemResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class DashboardStats {
	    total: number;
	    available: number;
	    error: number;
	    untested: number;
	    disabled: number;
	
	    static createFrom(source: any = {}) {
	        return new DashboardStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.available = source["available"];
	        this.error = source["error"];
	        this.untested = source["untested"];
	        this.disabled = source["disabled"];
	    }
	}
	export class FilterOptions {
	    providers: string[];
	    environments: string[];
	    groupNames: string[];
	
	    static createFrom(source: any = {}) {
	        return new FilterOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providers = source["providers"];
	        this.environments = source["environments"];
	        this.groupNames = source["groupNames"];
	    }
	}
	
	export class UpdateAPIKeyInput {
	    name: string;
	    provider: string;
	    environment: string;
	    groupName: string;
	    budgetTag: string;
	    baseUrl: string;
	    apiKey: string;
	    testEndpoint: string;
	    timeoutMs: number;
	    rateLimitRpm: number;
	    failureStrategy: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateAPIKeyInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.environment = source["environment"];
	        this.groupName = source["groupName"];
	        this.budgetTag = source["budgetTag"];
	        this.baseUrl = source["baseUrl"];
	        this.apiKey = source["apiKey"];
	        this.testEndpoint = source["testEndpoint"];
	        this.timeoutMs = source["timeoutMs"];
	        this.rateLimitRpm = source["rateLimitRpm"];
	        this.failureStrategy = source["failureStrategy"];
	    }
	}

}

