import {
    CloudDownload, CloudUpload, Download, Upload,
} from "@mui/icons-material";
import {
    Grid,
    Paper,
    Stack, Table, TableBody, TableCell,
    TableContainer, TableHead, TableRow, TableSortLabel, Tooltip
} from "@mui/material";
import { useEffect, useState } from "react";
import { EventsOff, EventsOn } from "../../wailsjs/runtime/runtime";

interface ConnectionSnapshot {
    id: string
    start_time: string
    network: string
    src: string,
    dst: string,
    download_total: number,
    upload_total: number,
    host: string,
    rule_mode: string,
    rule_type: string,
    proxy: string
}

interface AllDumpSnapshot {
    download_total: number
    upload_total: number
    connections?: ConnectionSnapshot[]
}

function descendingComparator<T>(a: T, b: T, orderBy: keyof T) {
    if (b[orderBy] < a[orderBy]) {
        return -1;
    }
    if (b[orderBy] > a[orderBy]) {
        return 1;
    }
    return 0;
}

type Order = 'asc' | 'desc';

function getComparator<Key extends keyof any>(order: Order, orderBy: Key,): (
    a: { [key in Key]: number | string },
    b: { [key in Key]: number | string },
) => number {
    return order === 'desc'
        ? (a, b) => descendingComparator(a, b, orderBy)
        : (a, b) => -descendingComparator(a, b, orderBy);
}

interface headCell {
    name: keyof ConnectionSnapshot
    value: string
}

const headCells: readonly headCell[] = [
    { name: "rule_type", value: "规则类型" },
    { name: "network", value: "网络" },
    { name: "src", value: "源地址" },
    { name: "dst", value: "目的地址" },
    { name: "host", value: "域名" },
    { name: "download_total", value: "下载大小" },
    { name: "upload_total", value: "上传大小" },
    { name: "rule_mode", value: "规则模式" },
    { name: "proxy", value: "代理" }
]

export default function CConnections() {
    const [downloadTraffic, setDownloadTraffic] = useState(0)
    const [uploadTraffic, setUploadTraffic] = useState(0)
    const [snapshot, setSnapshot] = useState<AllDumpSnapshot>()

    // order type
    const [order, setOrder] = useState<Order>('asc')
    // order by field
    const [orderBy, setOrderBy] = useState<keyof ConnectionSnapshot>('id')

    const handleRequestSort = (property: keyof ConnectionSnapshot) => {
        const isAsc = orderBy === property && order === 'asc'
        setOrder(isAsc ? 'desc' : 'asc')
        setOrderBy(property)
    }

    const formatBytes = (bytes?: number, decimals?: number) => {
        if (!bytes || bytes == 0) return '0 B';
        const k = 1000, dm = decimals || 2,
            sizes = ['B', 'KB', 'MB', 'GB'],
            i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
    }

    const formatRate = (bytes: number) => {
        return formatBytes(bytes) + "/s"
    }

    useEffect(() => {
        EventsOff('mini-ss-connection-traffic')
        EventsOff('mini-ss-connection-snapshot')

        EventsOn("mini-ss-connection-traffic", (...data: any) => {
            setDownloadTraffic(parseInt(data[0]))
            setUploadTraffic(parseInt(data[1]))
        })
        EventsOn("mini-ss-connection-snapshot", (data: any) => {
            setSnapshot(data as AllDumpSnapshot)
        })
    }, [])

    return (
        <Stack justifyContent="center" alignItems={"center"}>
            <Grid container spacing={2} justifyContent={"center"} alignItems="center">
                <Grid item>
                    <Stack padding={0.5} spacing={1} direction={"row"} >
                        <Stack direction={"row"} alignItems="center" justifyContent={"center"}>
                            <Tooltip title="总下载大小">
                                <CloudDownload color="success" />
                            </Tooltip>
                            {formatBytes(snapshot?.download_total)}
                        </Stack>
                        <Stack direction={"row"} alignItems="center" justifyContent={"center"}>
                            <Tooltip title="总上传大小">
                                <CloudUpload color="warning" />
                            </Tooltip>
                            {formatBytes(snapshot?.upload_total)}
                        </Stack>
                    </Stack>
                </Grid>
                <Grid item>
                    <Stack padding={0.5} spacing={1} direction={"row"}>
                        <Stack direction={"row"} alignItems="center" justifyContent={"center"}>
                            <Tooltip title="当前下载速率">
                                <Download color="success" />
                            </Tooltip>
                            {formatRate(downloadTraffic)}
                        </Stack>
                        <Stack direction={"row"} alignItems="center" justifyContent={"center"}>
                            <Tooltip title="当前上传速率">
                                <Upload color="warning" />
                            </Tooltip>
                            {formatRate(uploadTraffic)}
                        </Stack>
                    </Stack>
                </Grid>
            </Grid>
            <TableContainer component={Paper} sx={{ maxWidth: '100%', width: '100%', overflowX: 'scroll' }}>
                <Table
                    sx={{ width: "max-content" }}
                    size="small"
                    aria-label="a dense connection table"
                    stickyHeader>
                    <TableHead>
                        <TableRow>
                            {
                                headCells.map((val, index) =>
                                    <TableCell
                                        key={index}
                                        align="center"
                                    >
                                        <TableSortLabel
                                            active={orderBy == val.name}
                                            direction={orderBy === val.name ? order : 'asc'}
                                            onClick={() => handleRequestSort(val.name)}>
                                            {val.value}
                                        </TableSortLabel>
                                    </TableCell>
                                )
                            }
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {
                            snapshot?.connections &&
                            snapshot.connections
                                .sort(getComparator(order, orderBy))
                                .map((c, index) =>
                                    <TableRow hover key={index}>
                                        <TableCell align="center">{c.rule_type ? c.rule_type : 'N/A'}</TableCell>
                                        <TableCell align="center">{c.network}</TableCell>
                                        <TableCell align="center"><strong>{c.src ? c.src : 'N/A'}</strong></TableCell>
                                        <TableCell align="center"><strong>{c.dst}</strong></TableCell>
                                        <TableCell align="center"><strong>{c.host ? c.host : 'N/A'}</strong></TableCell>
                                        <TableCell align="center"><span style={{ color: '#0A870A' }}>{formatBytes(c.download_total)}</span></TableCell>
                                        <TableCell align="center"><span style={{ color: '#DCB123' }}>{formatBytes(c.upload_total)}</span></TableCell>
                                        <TableCell align="center">{c.rule_mode ? c.rule_mode : 'N/A'}</TableCell>
                                        <TableCell align="center">{c.proxy ? c.proxy : 'N/A'}</TableCell>
                                    </TableRow>
                                )
                        }
                    </TableBody>
                </Table>
            </TableContainer>
        </Stack >
    )
}
