import {
    Alert,
    Paper, Stack, Table, TableBody,
    TableCell, TableContainer, TableHead, TableRow
} from "@mui/material";
import { useEffect, useState } from "react";
import { config } from "../../wailsjs/go/models";

interface Props {
    cfgRules: config.Rules | undefined
}

interface Rule {
    type: string
    target: string
    proxy: string
    action: string
}

export default function CRules({ cfgRules }: Props) {

    const [rules, setRules] = useState<Rule[]>([])

    const ruleProxy = (action: string, proxy: string) => {
        switch (action) {
            case 'drop':
                return 'N/A'
            default:
                return proxy
        }
    }

    const convertToRules = () => {
        let _rules = new Array<Rule>()
        // convert to the array of rules
        if (cfgRules && cfgRules.mode == 'match' && cfgRules.match) {
            // domain 
            if (cfgRules.match.domain) {
                for (const rule of cfgRules.match.domain) {
                    for (const item of rule.value) {
                        _rules.push({ type: 'Domain', target: item, proxy: ruleProxy(rule.action, rule.proxy), action: rule.action })
                    }
                }
            }
            // domain-keyword   
            if (cfgRules.match.domain_keyword) {
                for (const rule of cfgRules.match.domain_keyword) {
                    for (const item of rule.value) {
                        _rules.push({ type: 'Domain-Keyword', target: item, proxy: ruleProxy(rule.action, rule.proxy), action: rule.action })
                    }
                }
            }
            // domain-suffix
            if (cfgRules.match.domain_suffix) {
                for (const rule of cfgRules.match.domain_suffix) {
                    for (const item of rule.value) {
                        _rules.push({ type: 'Domain-Suffix', target: item, proxy: ruleProxy(rule.action, rule.proxy), action: rule.action })
                    }
                }
            }
            // geoip
            if (cfgRules.match.geoip) {
                for (const rule of cfgRules.match.geoip) {
                    for (const item of rule.value) {
                        _rules.push({ type: 'GeoIP', target: item, proxy: ruleProxy(rule.action, rule.proxy), action: rule.action })
                    }
                }
            }
            // ipcidr
            if (cfgRules.match.ipcidr) {
                for (const rule of cfgRules.match.ipcidr) {
                    for (const item of rule.value) {
                        _rules.push({ type: 'IP-CIDR', target: item, proxy: ruleProxy(rule.action, rule.proxy), action: rule.action })
                    }
                }
            }

            if (cfgRules.match.others) {
                _rules.push({ type: "Others", target: "*", proxy: cfgRules.match.others, action: "accept" })
            }

            setRules(_rules)
        }
    }

    useEffect(() => {
        convertToRules()
    }, [cfgRules])


    return (
        <Stack spacing={1}>

            <Alert severity="success">
                匹配规则: <strong>{cfgRules ? cfgRules.mode : 'global'}</strong>
            </Alert>

            <TableContainer component={Paper}>
                <Table sx={{ minWidth: 650 }} size="small" aria-label="a dense rule table">
                    <TableHead>
                        <TableRow>
                            <TableCell align="center">类型</TableCell>
                            <TableCell align="center">目标</TableCell>
                            <TableCell align="center">代理</TableCell>
                            <TableCell align="center">行为</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {
                            rules.map((row, index) =>
                                <TableRow
                                    hover
                                    key={index}
                                    sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
                                >
                                    <TableCell align="center">{row.type}</TableCell>
                                    <TableCell align="center">{row.target}</TableCell>
                                    <TableCell align="center">{row.proxy}</TableCell>
                                    <TableCell align="center">{row.action}</TableCell>
                                </TableRow>
                            )
                        }
                    </TableBody>
                </Table>
            </TableContainer>
        </Stack>
    )
}