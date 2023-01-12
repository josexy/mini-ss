import {
    Button, IconButton,
    InputAdornment, Stack, TextField, Tooltip
} from "@mui/material";
import { useState } from "react";
import AceEditor from "react-ace";

import "ace-builds/src-noconflict/mode-json";
import "ace-builds/src-noconflict/theme-chrome";
import "ace-builds/src-noconflict/ext-language_tools";
import { ImportExport, UploadFile } from "@mui/icons-material";
import { config } from "../../wailsjs/go/models";
import {
    ExportCurrentConfig, GetJsonConfigContent,
    GetJsonConfigFilePath, LoadConfig, LoadConfigFile
} from "../../wailsjs/go/main/App";

interface Props {
    setCfgJson: React.Dispatch<React.SetStateAction<config.JsonConfig | undefined>>
    showToast: (message: string, type: string) => void
}

export default function CConfig({ setCfgJson, showToast }: Props) {
    const [jsonConfigContent, setJsonConfigContent] = useState<string>("")
    const [jsonConfigFile, setJsonConfigFile] = useState<string>("")

    const loadConfig = () => {
        const configFile = jsonConfigFile.trim()

        let promise: Promise<config.JsonConfig>
        // load config from path directly
        if (configFile !== "") {
            promise = LoadConfigFile(configFile)
        } else {
            promise = LoadConfig()
        }
        promise.then(c => {
            setCfgJson(c)

            GetJsonConfigFilePath().then(path => {
                setJsonConfigFile(path)
            })

            GetJsonConfigContent().then(content => {
                setJsonConfigContent(content)
            })
            showToast("加载配置文件成功！", 'success')
        }).catch(err => {
            showToast(`错误: ${err}`, 'error')
        })
    }

    const exportCurrentConfig = () => {
        ExportCurrentConfig().then(() => {
            showToast('导出配置文件成功！', 'success')
        }).catch(err => {
            showToast(`错误: ${err}`, 'error')
        })
    }

    return (
        <Stack spacing={1}>
            <TextField variant="standard" size="small"
                value={jsonConfigFile}
                onChange={(e: any) => setJsonConfigFile(e.target.value)}
                InputProps={{
                    endAdornment: (
                        <InputAdornment position="end">
                            <Tooltip title="加载">
                                <IconButton edge="end"
                                    onClick={() => loadConfig()}>
                                    <UploadFile />
                                </IconButton >
                            </Tooltip>
                        </InputAdornment>
                    )
                }}
            />
            <AceEditor
                readOnly
                wrapEnabled
                style={{ "border": "1px solid lightgray" }}
                width="100%"
                mode="json"
                theme="chrome"
                name="json_config"
                fontSize={14}
                showPrintMargin={false}
                showGutter={true}
                highlightActiveLine={true}
                value={jsonConfigContent}
                onChange={(value: string) => setJsonConfigContent(value)}
                onLoad={editorInstance => {
                    editorInstance.container.style.resize = "vertical";
                    document.addEventListener("mouseup", e => (
                        editorInstance.resize()
                    ));
                }}
                setOptions={{
                    enableBasicAutocompletion: false,
                    enableLiveAutocompletion: false,
                    enableSnippets: false,
                    showLineNumbers: true,
                }}
                editorProps={{ $blockScrolling: true }}
            />
            <Button startIcon={<ImportExport />}
                onClick={() => exportCurrentConfig()}>
                导出当前配置
            </Button>
        </Stack >
    )
}