import { Clear } from "@mui/icons-material";
import { Button, Stack, TextField } from "@mui/material";
import { useEffect, useState } from "react";

import { EventsOff, EventsOn } from "../../wailsjs/runtime/runtime"

export default function CLogger() {

    const [loggerMessage, setLoggerMessage] = useState('')

    const clearLogger = () => {
        setLoggerMessage('')
    }

    useEffect(() => {
        EventsOff('mini-ss-logger')
        EventsOn("mini-ss-logger", (data: any) => {
            setLoggerMessage(old => old + data)
        })
    }, [])

    return (
        <Stack spacing={1}>
            <TextField size="small" label="日志内容" variant="outlined"
                multiline minRows={10} maxRows={20}
                InputProps={{ readOnly: true }}
                value={loggerMessage}
            />
            <Button color="error"
                startIcon={<Clear />}
                onClick={() => clearLogger()}>清空</Button>
        </Stack >
    )
}   