type Fields = Record<string, unknown>

function emit(level: string, msg: string, fields?: Fields): void {
    const entry = {
        time: new Date().toISOString(),
        level,
        msg,
        ...fields,
    }
    const line = JSON.stringify(entry)

    if (level === 'error') {
        console.error(line)
    } else {
        console.log(line)
    }
}

export const logger = {
    info: (msg: string, fields?: Fields) => emit('info', msg, fields),
    warn: (msg: string, fields?: Fields) => emit('warn', msg, fields),
    error: (msg: string, fields?: Fields) => emit('error', msg, fields),
}
