import { useEffect, useState } from 'react';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemText from '@mui/material/ListItemText';
import {
    Alert, Badge, Box, Button, ButtonGroup, Checkbox, FormControl,
    FormControlLabel, FormGroup, Grid,
    IconButton, InputAdornment, InputLabel,
    MenuItem, Paper, Select,
    Stack, TextField, Tooltip
} from '@mui/material';
import {
    AddServerConfig,
    DeleteServerConfig,
    GetSupportCipherMethods, GetSupportKcpCrypts,
    GetSupportKcpModes, GetSupportTransportTypes,
    UpdateServerConfig, Ping, TcpPing
} from '../../wailsjs/go/main/App';
import { config } from '../../wailsjs/go/models';
import {
    Add, ChangeCircle, Create, Delete,
    NetworkCheck,
    NetworkPing,
    Save, Visibility, VisibilityOff, Wifi
} from '@mui/icons-material';
import useMap from '../hooks/useMap';

interface Props {
    cfgServer?: config.ServerJsonConfig[]
    showToast: (message: string, type: string) => void
}

interface CheckStatus {
    // proxy network result: RTT, timeout
    result?: string
    // proxy network status: success, fail, unknown
    status?: string
}

function NetworkStatus({ result, status }: CheckStatus) {

    // 200ms
    const threshold = 200
    const units = ["µs", "ms", "s", "h"]

    const calcStatus = () => {
        // timeout
        if (status === 'fail') return 'error'
        // success
        if (status === 'success') {
            if (result) {
                let index = -1
                for (const unit of units) {
                    index = result.indexOf(unit)
                    if (index !== -1) break
                }
                if (index !== -1) {
                    const rtt = result.substring(0, index)
                    if (parseInt(rtt) >= threshold) {
                        return 'warning'
                    }
                }
            }
            return 'success'
        }
        // unknown
        return 'disabled'
    }
    return (
        <div>
            <small style={{ color: 'gray' }}>
                {result}
            </small>
            <Badge color="success" badgeContent={0}>
                <Wifi fontSize='small'
                    color={calcStatus()} />
            </Badge>
        </div>
    )
}

export default function CServers({ cfgServer, showToast }: Props) {
    const [serverCfg, setServerCfg] = useState<config.ServerJsonConfig[]>([])

    // enable/disable
    const [disable, setDisable] = useState(false)

    const [name, setName] = useState('')
    const [addr, setAddr] = useState('')
    const [password, setPassword] = useState('')
    const [method, setMethod] = useState('none')
    const [transportType, setTransportType] = useState('default')
    const [type, setType] = useState(false)

    // ssr option
    const [ssrProto, setProto] = useState('')
    const [ssrProtoParam, setProtoParam] = useState('')
    const [ssrObfs, setObfs] = useState('')
    const [ssrObfsParam, setObfsParam] = useState('')

    // ws option
    const [wsHost, setWsHost] = useState('')
    const [wsPath, setWsPath] = useState('')
    const [wsCompress, setWsCompress] = useState(false)
    const [wsTLS, setWsTLS] = useState(false)

    // kcp option
    const [kcpCrypt, setKcpCrypt] = useState('none')
    const [kcpKey, setKcpKey] = useState('')
    const [kcpMode, setKcpMode] = useState('normal')
    const [kcpCompress, setKcpCompress] = useState(false)
    const [kcpConns, setKcpConns] = useState(3)

    // obfs option
    const [obfsHost, setObfsHost] = useState('')
    const [obfsTLS, setObfsTLS] = useState(false)

    // quic option
    const [quicConns, setQuicConns] = useState(3)

    const [supportMethods, setSupportMethods] = useState<string[]>([])
    const [supportTransportTypes, setSupportTransportTypes] = useState<string[]>([])
    const [supportKcpCrypts, setSupportKcpCrypts] = useState<string[]>([])
    const [supportKcpModes, setSupportKcpModes] = useState<string[]>([])
    const [listProxies, setListProxies] = useState<string[]>([])

    const [passwordVisible, setPasswordVisible] = useState(false)
    const [listItemSelIndex, setListItemSelIndex] = useState(-1)

    const [checkStatus, checkStatusActions] = useMap<string, CheckStatus>()

    useEffect(() => {
        GetSupportCipherMethods().then(list => setSupportMethods(list))
        GetSupportTransportTypes().then(list => setSupportTransportTypes(list))
        GetSupportKcpCrypts().then(list => setSupportKcpCrypts(list))
        GetSupportKcpModes().then(list => setSupportKcpModes(list))
    }, [])

    // when the config loaded, udpate local object
    useEffect(() => {
        if (cfgServer) {
            let list = new Array<string>()
            cfgServer.forEach((val: config.ServerJsonConfig) => {
                list.push(val.name)
            })
            // copy old root config object
            setServerCfg(cfgServer)
            setListProxies(list)
        }
    }, [cfgServer])

    const newProxy = () => {
        resetCurrentStatus()
        setListItemSelIndex(-1)
    }

    const addProxy = () => {
        const _name = name.trim()
        if (_name === '') {
            showToast('代理服务名称不能为空！', 'error')
            return
        }
        if (-1 !== listProxies.indexOf(_name)) {
            showToast('代理服务已经存在', 'error')
            return
        }
        let value = newServerConfig()
        // add config object
        setListProxies(old => [...old, _name])
        setServerCfg(old => [...old, value])
        AddServerConfig(value).then(() => showToast('添加代理服务成功！', 'success'))
        // reset input status
        newProxy()

        // add network status
        checkStatusActions.set(_name, { result: '', status: 'unknown' })
    }

    const saveProxy = () => {
        const _name = name.trim()
        if (_name === '') {
            showToast('代理服务名称不能为空！', 'error')
            return
        }
        let index = listProxies.indexOf(_name)
        if (-1 === index) {
            showToast('代理服务不存在！', 'error')
            return
        }
        for (let i = 0; i < serverCfg.length; i++) {
            if (serverCfg[i].name === _name) {
                index = i
                break
            }
        }
        if (index === -1) {
            return
        }
        let value = newServerConfig()
        // copy object and update state
        let copyCfg = [...serverCfg]
        copyCfg[index] = value
        setServerCfg(copyCfg)
        UpdateServerConfig(value).then(() => showToast('保存代理服务成功！', 'success'))
    }

    const deleteProxy = () => {
        const _name = name.trim()
        if (-1 === listProxies.indexOf(_name)) {
            showToast('代理服务不存在！', 'error')
            return
        }
        setListProxies(old => old.filter((val, _) => val != _name))
        DeleteServerConfig(_name).then(() => showToast('删除代理服务成功！', 'success'))

        // reset 
        newProxy()
        // delete network status
        checkStatusActions.remove(_name)
    }

    const resetWsStatus = (v?: config.WsOption) => {
        if (v) {
            setWsHost(v.host)
            setWsPath(v.path)
            setWsCompress(v.compress)
            setWsTLS(v.tls)
            return
        }
        setWsHost('')
        setWsPath('')
        setWsCompress(false)
        setWsTLS(false)
    }

    const resetKcpStatus = (v?: config.KcpOption) => {
        if (v) {
            setKcpCrypt(v.crypt)
            setKcpKey(v.key)
            setKcpMode(v.mode)
            setKcpCompress(v.compress)
            setKcpConns(v.conns)
            return
        }
        setKcpCrypt('none')
        setKcpKey('')
        setKcpMode('normal')
        setKcpCompress(false)
        setKcpConns(3)
    }

    const resetQuicStatus = (v?: config.QuicOption) => {
        if (v) {
            setQuicConns(v.conns)
            return
        }
        setQuicConns(3)
    }

    const resetObfsStatus = (v?: config.ObfsOption) => {
        if (v) {
            setObfsHost(v.host)
            setObfsTLS(v.tls)
            return
        }
        setObfsHost('')
        setObfsTLS(false)
    }

    const resetSSRStatus = (v?: config.SSROption) => {
        if (v) {
            setProto(v.protocol)
            setProtoParam(v.protocol_param)
            setObfs(v.obfs)
            setObfsParam(v.obfs_param)
            return
        }
        setProto('')
        setProtoParam('')
        setObfs('')
        setObfsParam('')
    }

    const resetCurrentStatus = (v?: config.ServerJsonConfig) => {
        if (v) {
            setDisable(v.disable)
            setName(v.name)
            setAddr(v.addr)
            setPassword(v.password)
            setMethod(v.method)
            setTransportType(v.transport)
            setType(v.type === 'ssr')
        } else {
            setDisable(false)
            setName('')
            setAddr('')
            setPassword('')
            setMethod('none')
            setTransportType('default')
            setType(false)
        }

        resetWsStatus()
        resetKcpStatus()
        resetQuicStatus()
        resetObfsStatus()
        resetSSRStatus()

        if (v) {
            switch (v.transport) {
                case "ws":
                    resetWsStatus(v.ws)
                    break;
                case "kcp":
                    resetKcpStatus(v.kcp)
                    break;
                case "quic":
                    resetQuicStatus(v.quic)
                    break;
                case "obfs":
                    resetObfsStatus(v.obfs)
                    break;
                default:
                    resetSSRStatus(v.ssr)
                    break;
            }
        }
    }

    const newServerConfig = () => {
        let cfg = config.ServerJsonConfig.createFrom({
            disable: disable,
            name: name,
            addr: addr,
            type: type ? 'ssr' : '',
            password: password,
            method: method,
            transport: transportType,
        })
        switch (cfg.transport) {
            case "ws":
                cfg.ws = newWsOption()
                break;
            case "quic":
                cfg.quic = newQuicOption()
                break;
            case "kcp":
                cfg.kcp = newKcpOption()
                break;
            case "obfs":
                cfg.obfs = newObfsOption()
                break;
            case "default":
                // enable ssr
                if (type) {
                    cfg.ssr = newSSROption()
                }
                break;
        }
        return cfg
    }

    const newSSROption = () => {
        return config.SSROption.createFrom({
            protocol: ssrProto,
            protocol_param: ssrProtoParam,
            obfs: ssrObfs,
            obfs_param: ssrObfsParam,
        })
    }

    const newWsOption = () => {
        return config.WsOption.createFrom({
            host: wsHost,
            path: wsPath,
            compress: wsCompress,
            tls: wsTLS,
        })
    }

    const newQuicOption = () => {
        return config.QuicOption.createFrom({
            conns: quicConns,
        })
    }

    const newKcpOption = () => {
        return config.KcpOption.createFrom({
            key: kcpKey,
            crypt: kcpCrypt,
            mode: kcpMode,
            conns: kcpConns,
            compress: kcpCompress,
        })
    }

    const newObfsOption = () => {
        return config.ObfsOption.createFrom({
            host: obfsHost,
            tls: obfsTLS,
        })
    }

    // select list item
    const listItemClick = (name: string, index: number) => {
        for (const v of serverCfg) {
            if (name === v.name) {
                resetCurrentStatus(v)
                setListItemSelIndex(index)
                break
            }
        }
    }

    // check network status: icmp ping and tcp ping
    const checkNetworkStatus = (type: string) => {
        let handleFunc: (arg1: string) => Promise<string>
        switch (type) {
            case 'ping':
                handleFunc = Ping;
                break;
            case 'tcping':
                handleFunc = TcpPing;
                break;
            default:
                return;
        }
        if (serverCfg) {
            for (const s of serverCfg) {
                const proxy = s.addr.trim()
                if (proxy === '') {
                    continue
                }
                handleFunc(proxy).then(res => {
                    console.log(res);
                    checkStatusActions.set(s.name, { result: res, status: 'success' })
                }).catch((err) => {
                    console.log(s.name, err);
                    checkStatusActions.set(s.name, { result: 'timeout', status: 'fail' })
                })
            }
        }
    }

    const pingClick = () => { checkNetworkStatus('ping') }
    const tcpPingClick = () => { checkNetworkStatus('tcping') }

    return (
        <Paper>
            <Stack
                padding={2}
                direction={"row"}
                spacing={2}>
                <Grid container spacing={2}>
                    <Grid item sm={3}>
                        <Box marginBottom={1}>
                            <Paper>
                                <Stack direction={"row"} justifyContent="center" alignItems={"center"}>
                                    <ButtonGroup
                                        size='small' variant="outlined"
                                        color='success' >
                                        <Tooltip title='icmp ping'><IconButton onClick={() => pingClick()} size='small'><NetworkPing /></IconButton></Tooltip>
                                        <Tooltip title='tcp ping'><IconButton onClick={() => tcpPingClick()} size='small'><NetworkCheck /></IconButton></Tooltip>
                                    </ButtonGroup>
                                </Stack>
                            </Paper>
                        </Box>

                        <Paper>
                            <List dense sx={{ maxHeight: 400, overflow: 'auto' }}>
                                {
                                    listProxies.map((val, index) =>
                                        <Tooltip title={val} key={index}>
                                            <ListItem
                                                disablePadding
                                                onClick={() => listItemClick(val, index)}>
                                                <ListItemButton
                                                    sx={{
                                                        "height": "25px",
                                                        "&.Mui-selected": { backgroundColor: '#AED6F1' },
                                                    }}
                                                    selected={index === listItemSelIndex}>
                                                    <ListItemText primary={val} />
                                                    <NetworkStatus
                                                        result={checkStatus.get(val)?.result}
                                                        status={checkStatus.get(val)?.status} />
                                                </ListItemButton>
                                            </ListItem>
                                        </Tooltip>
                                    )
                                }
                            </List>
                        </Paper>
                    </Grid>

                    <Grid item sm={9}>
                        <Paper>
                            <Stack padding={2}>
                                <Alert sx={{ height: 35, alignItems: "center" }} severity={disable ? "error" : "success"}>
                                    节点状态: {disable ? "禁止" : "开启"}
                                    <IconButton size='small' color='inherit'
                                        aria-label='switch' onClick={() => setDisable(!disable)}>
                                        <ChangeCircle />
                                    </IconButton>

                                    SSR: {type ? "支持" : "不支持"}
                                    <IconButton size='small' color='inherit'
                                        aria-label='switch' onClick={() => setType(!type)}>
                                        <ChangeCircle />
                                    </IconButton>
                                </Alert>

                                <TextField size='small' label="名称" variant="standard"
                                    value={name} onChange={(e: any) => setName(e.target.value)} />
                                <TextField size='small' label="地址" variant="standard"
                                    value={addr}
                                    onChange={(e: any) => setAddr(e.target.value)} />
                                <TextField size='small' label="密码" variant="standard"
                                    value={password}
                                    type={passwordVisible ? "text" : "password"}
                                    onChange={(e: any) => setPassword(e.target.value)}
                                    InputProps={{
                                        endAdornment: (
                                            <InputAdornment position="end">
                                                <IconButton edge="end"
                                                    onClick={() => setPasswordVisible(!passwordVisible)}
                                                >
                                                    {passwordVisible ? <Visibility /> : <VisibilityOff />}
                                                </IconButton >
                                            </InputAdornment>
                                        )
                                    }}
                                />
                                <FormControl size='small' margin='dense'>
                                    <InputLabel id="method-type">加密方法</InputLabel>
                                    <Select
                                        labelId="method-type"
                                        value={method}
                                        label="加密方法"
                                        onChange={(e: any) => setMethod(e.target.value)}
                                    >
                                        {
                                            supportMethods.map((val, index) => <MenuItem key={index} value={val}>{val}</MenuItem>)
                                        }
                                    </Select>
                                </FormControl>

                                <FormControl size='small' margin='dense'>
                                    <InputLabel id="transport-type">传输类型</InputLabel>
                                    <Select
                                        labelId="transport-type"
                                        value={transportType}
                                        label="传输类型"
                                        onChange={(e: any) => setTransportType(e.target.value)}
                                    >
                                        {
                                            supportTransportTypes.map((val, index) => <MenuItem key={index} value={val}>{val}</MenuItem>)
                                        }
                                    </Select>
                                </FormControl>

                                {(() => {
                                    switch (transportType) {
                                        case "default":
                                            if (type) {
                                                return <Stack sx={{ display: "" }}>
                                                    <FormGroup row>
                                                        <TextField size='small' label="协议" variant="standard"
                                                            value={ssrProto} onChange={(e: any) => setProto(e.target.value)} />
                                                        <TextField size='small' label="协议参数" variant="standard"
                                                            value={ssrProtoParam} onChange={(e: any) => setProtoParam(e.target.value)} />
                                                    </FormGroup>
                                                    <FormGroup row>
                                                        <TextField size='small' label="混淆" variant="standard"
                                                            value={ssrObfs} onChange={(e: any) => setObfs(e.target.value)} />
                                                        <TextField size='small' label="混淆参数" variant="standard"
                                                            value={ssrObfsParam} onChange={(e: any) => setObfsParam(e.target.value)} />
                                                    </FormGroup>
                                                </Stack>
                                            }
                                            break
                                        case "ws":
                                            return <Stack sx={{ display: "" }}>
                                                <TextField size='small' label="host" variant="standard"
                                                    value={wsHost} onChange={(e: any) => setWsHost(e.target.value)} />
                                                <TextField size='small' label="path" variant="standard"
                                                    value={wsPath} onChange={(e: any) => setWsPath(e.target.value)} />
                                                <FormGroup row>
                                                    <FormControlLabel control={<Checkbox
                                                        checked={wsCompress}
                                                        onChange={(_, checked: boolean) => setWsCompress(checked)} />} label="compress" />
                                                    <FormControlLabel control={<Checkbox
                                                        checked={wsTLS}
                                                        onChange={(_, checked: boolean) => setWsTLS(checked)} />} label="tls" />
                                                </FormGroup>
                                            </Stack>
                                        case "obfs":
                                            return <Stack sx={{ display: "" }}>
                                                <TextField size='small' label="host" variant="standard"
                                                    value={obfsHost} onChange={(e: any) => setObfsHost(e.target.value)} />
                                                <FormGroup row>
                                                    <FormControlLabel control={<Checkbox
                                                        checked={obfsTLS}
                                                        onChange={(_, checked: boolean) => setObfsTLS(checked)} />} label="tls" />
                                                </FormGroup>
                                            </Stack>
                                        case "quic":
                                            return <Stack sx={{ display: "" }}>
                                                <TextField size='small' type="number" label="conns" variant="standard"
                                                    value={quicConns} onChange={(e: any) => setQuicConns(e.target.value)} />
                                            </Stack>
                                        case "kcp":
                                            return <Stack sx={{ display: "" }}>
                                                <TextField size='small' label="key" variant="standard"
                                                    value={kcpKey} onChange={(e: any) => setKcpKey(e.target.value)} />
                                                <TextField size='small' type="number" label="conns" variant="standard"
                                                    value={kcpConns} onChange={(e: any) => setKcpConns(e.target.value)} />
                                                <Grid container spacing={1} alignItems="center">
                                                    <Grid item>
                                                        <FormControl size='small' margin='dense'>
                                                            <InputLabel id="kcp-crypt">crypt</InputLabel>
                                                            <Select
                                                                labelId="kcp-crypt"
                                                                value={kcpCrypt}
                                                                label="crypt"
                                                                onChange={(e: any) => setKcpCrypt(e.target.value)}
                                                            >
                                                                {
                                                                    supportKcpCrypts.map((val, index) => <MenuItem key={index} value={val}>{val}</MenuItem>)
                                                                }
                                                            </Select>
                                                        </FormControl>
                                                    </Grid>
                                                    <Grid item>
                                                        <FormControl size='small' margin='dense'>
                                                            <InputLabel id="kcp-mode">mode</InputLabel>
                                                            <Select
                                                                labelId="kcp-mode"
                                                                value={kcpMode}
                                                                label="mode"
                                                                onChange={(e: any) => setKcpMode(e.target.value)}
                                                            >
                                                                {
                                                                    supportKcpModes.map((val, index) => <MenuItem key={index} value={val}>{val}</MenuItem>)
                                                                }
                                                            </Select>
                                                        </FormControl>
                                                    </Grid>
                                                    <Grid item>
                                                        <FormGroup row>
                                                            <FormControlLabel control={<Checkbox
                                                                checked={kcpCompress}
                                                                onChange={(_, checked: boolean) => setKcpCompress(checked)} />} label="compress" />
                                                        </FormGroup>
                                                    </Grid>
                                                </Grid>
                                            </Stack>
                                    }
                                })()}

                                <Stack justifyContent={"center"} direction={"row"} paddingTop={1} spacing={1}>
                                    <Button color='primary' variant='outlined'
                                        startIcon={<Create />}
                                        onClick={() => newProxy()}>新建</Button>
                                    <Button color='info' variant='outlined'
                                        onClick={() => addProxy()}
                                        startIcon={<Add />}>添加</Button>
                                    <Button color="success" variant="outlined"
                                        startIcon={<Save />}
                                        onClick={() => saveProxy()}>保存</Button>
                                    <Button color="error" variant="outlined"
                                        startIcon={<Delete />}
                                        onClick={() => deleteProxy()}>删除</Button>
                                </Stack>
                            </Stack>
                        </Paper>
                    </Grid>
                </Grid>
            </Stack>
        </Paper>
    )
}