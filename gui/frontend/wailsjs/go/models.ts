export namespace config {
	
	export class IPCidr {
	    resolve?: boolean;
	    proxy: string;
	    action: string;
	    value: string[];
	
	    static createFrom(source: any = {}) {
	        return new IPCidr(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resolve = source["resolve"];
	        this.proxy = source["proxy"];
	        this.action = source["action"];
	        this.value = source["value"];
	    }
	}
	export class GeoIP {
	    resolve?: boolean;
	    proxy: string;
	    action: string;
	    value: string[];
	
	    static createFrom(source: any = {}) {
	        return new GeoIP(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resolve = source["resolve"];
	        this.proxy = source["proxy"];
	        this.action = source["action"];
	        this.value = source["value"];
	    }
	}
	export class DomainSuffix {
	    proxy: string;
	    action: string;
	    value: string[];
	
	    static createFrom(source: any = {}) {
	        return new DomainSuffix(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxy = source["proxy"];
	        this.action = source["action"];
	        this.value = source["value"];
	    }
	}
	export class DomainKeyword {
	    proxy: string;
	    action: string;
	    value: string[];
	
	    static createFrom(source: any = {}) {
	        return new DomainKeyword(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxy = source["proxy"];
	        this.action = source["action"];
	        this.value = source["value"];
	    }
	}
	export class Domain {
	    proxy: string;
	    action: string;
	    value: string[];
	
	    static createFrom(source: any = {}) {
	        return new Domain(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxy = source["proxy"];
	        this.action = source["action"];
	        this.value = source["value"];
	    }
	}
	export class Match {
	    others?: string;
	    domain?: Domain[];
	    domain_keyword?: DomainKeyword[];
	    domain_suffix?: DomainSuffix[];
	    geoip?: GeoIP[];
	    ipcidr?: IPCidr[];
	
	    static createFrom(source: any = {}) {
	        return new Match(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.others = source["others"];
	        this.domain = this.convertValues(source["domain"], Domain);
	        this.domain_keyword = this.convertValues(source["domain_keyword"], DomainKeyword);
	        this.domain_suffix = this.convertValues(source["domain_suffix"], DomainSuffix);
	        this.geoip = this.convertValues(source["geoip"], GeoIP);
	        this.ipcidr = this.convertValues(source["ipcidr"], IPCidr);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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
	export class Rules {
	    mode: string;
	    direct_to?: string;
	    global_to?: string;
	    match?: Match;
	
	    static createFrom(source: any = {}) {
	        return new Rules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.direct_to = source["direct_to"];
	        this.global_to = source["global_to"];
	        this.match = this.convertValues(source["match"], Match);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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
	export class FakeDnsOption {
	    listen: string;
	    nameservers: string[];
	
	    static createFrom(source: any = {}) {
	        return new FakeDnsOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listen = source["listen"];
	        this.nameservers = source["nameservers"];
	    }
	}
	export class TunOption {
	    name: string;
	    cidr: string;
	    mtu: number;
	
	    static createFrom(source: any = {}) {
	        return new TunOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.cidr = source["cidr"];
	        this.mtu = source["mtu"];
	    }
	}
	export class LocalConfig {
	    socks_addr?: string;
	    http_addr?: string;
	    socks_auth?: string;
	    http_auth?: string;
	    mixed_addr?: string;
	    tcp_tun_addr?: string[];
	    system_proxy?: boolean;
	    enable_tun?: boolean;
	    tun?: TunOption;
	    fake_dns?: FakeDnsOption;
	
	    static createFrom(source: any = {}) {
	        return new LocalConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.socks_addr = source["socks_addr"];
	        this.http_addr = source["http_addr"];
	        this.socks_auth = source["socks_auth"];
	        this.http_auth = source["http_auth"];
	        this.mixed_addr = source["mixed_addr"];
	        this.tcp_tun_addr = source["tcp_tun_addr"];
	        this.system_proxy = source["system_proxy"];
	        this.enable_tun = source["enable_tun"];
	        this.tun = this.convertValues(source["tun"], TunOption);
	        this.fake_dns = this.convertValues(source["fake_dns"], FakeDnsOption);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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
	export class SSROption {
	    protocol: string;
	    protocol_param?: string;
	    obfs: string;
	    obfs_param?: string;
	
	    static createFrom(source: any = {}) {
	        return new SSROption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.protocol = source["protocol"];
	        this.protocol_param = source["protocol_param"];
	        this.obfs = source["obfs"];
	        this.obfs_param = source["obfs_param"];
	    }
	}
	export class GrpcOption {
	    hostname?: string;
	    key_path?: string;
	    cert_path?: string;
	    ca_path?: string;
	    tls?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GrpcOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hostname = source["hostname"];
	        this.key_path = source["key_path"];
	        this.cert_path = source["cert_path"];
	        this.ca_path = source["ca_path"];
	        this.tls = source["tls"];
	    }
	}
	export class QuicOption {
	    conns: number;
	
	    static createFrom(source: any = {}) {
	        return new QuicOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conns = source["conns"];
	    }
	}
	export class ObfsOption {
	    host?: string;
	
	    static createFrom(source: any = {}) {
	        return new ObfsOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	    }
	}
	export class WsOption {
	    path: string;
	    host?: string;
	    compress?: boolean;
	    tls?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WsOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.host = source["host"];
	        this.compress = source["compress"];
	        this.tls = source["tls"];
	    }
	}
	export class KcpOption {
	    crypt: string;
	    key: string;
	    mode: string;
	    compress?: boolean;
	    conns: number;
	
	    static createFrom(source: any = {}) {
	        return new KcpOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.crypt = source["crypt"];
	        this.key = source["key"];
	        this.mode = source["mode"];
	        this.compress = source["compress"];
	        this.conns = source["conns"];
	    }
	}
	export class ServerConfig {
	    disable?: boolean;
	    type?: string;
	    name: string;
	    addr: string;
	    password: string;
	    method: string;
	    transport: string;
	    udp?: boolean;
	    kcp?: KcpOption;
	    ws?: WsOption;
	    obfs?: ObfsOption;
	    quic?: QuicOption;
	    grpc?: GrpcOption;
	    ssr?: SSROption;
	
	    static createFrom(source: any = {}) {
	        return new ServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.disable = source["disable"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.addr = source["addr"];
	        this.password = source["password"];
	        this.method = source["method"];
	        this.transport = source["transport"];
	        this.udp = source["udp"];
	        this.kcp = this.convertValues(source["kcp"], KcpOption);
	        this.ws = this.convertValues(source["ws"], WsOption);
	        this.obfs = this.convertValues(source["obfs"], ObfsOption);
	        this.quic = this.convertValues(source["quic"], QuicOption);
	        this.grpc = this.convertValues(source["grpc"], GrpcOption);
	        this.ssr = this.convertValues(source["ssr"], SSROption);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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
	export class Config {
	    server?: ServerConfig[];
	    local?: LocalConfig;
	    color?: boolean;
	    verbose?: boolean;
	    verbose_level?: number;
	    iface?: string;
	    auto_detect_iface?: boolean;
	    rules?: Rules;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = this.convertValues(source["server"], ServerConfig);
	        this.local = this.convertValues(source["local"], LocalConfig);
	        this.color = source["color"];
	        this.verbose = source["verbose"];
	        this.verbose_level = source["verbose_level"];
	        this.iface = source["iface"];
	        this.auto_detect_iface = source["auto_detect_iface"];
	        this.rules = this.convertValues(source["rules"], Rules);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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

export namespace main {
	
	export class Config {
	    path: string;
	    value?: config.Config;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.value = this.convertValues(source["value"], config.Config);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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

