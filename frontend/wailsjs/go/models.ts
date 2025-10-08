export namespace main {
	
	export class FileContentResult {
	    path: string;
	    content: string;
	    size: number;
	    isBinary: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new FileContentResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.content = source["content"];
	        this.size = source["size"];
	        this.isBinary = source["isBinary"];
	        this.error = source["error"];
	    }
	}
	export class FileNode {
	    name: string;
	    path: string;
	    relPath: string;
	    isDir: boolean;
	    children?: FileNode[];
	    isGitignored: boolean;
	    isCustomIgnored: boolean;
	    size: number;
	    isBinary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.relPath = source["relPath"];
	        this.isDir = source["isDir"];
	        this.children = this.convertValues(source["children"], FileNode);
	        this.isGitignored = source["isGitignored"];
	        this.isCustomIgnored = source["isCustomIgnored"];
	        this.size = source["size"];
	        this.isBinary = source["isBinary"];
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
	export class Job {
	    id: string;
	    type: string;
	    status: string;
	    progress: number;
	    error: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    startedAt: any;
	    // Go type: time
	    completedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.error = source["error"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], null);
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

}

