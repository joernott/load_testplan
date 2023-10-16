const childProcess = require('child_process')
const os = require('os')
const process = require('process')

const VERSION = '9d1c08f43ae724b801ca871dca96d9ac6a9d8e0f'

function chooseBinary() {
    const platform = os.platform()
    const arch = os.arch()

    if (platform === 'linux' && arch === 'x64') {
        return `main-linux-amd64-${VERSION}`
    }
    if (platform === 'linux' && arch === 'arm64') {
        return `main-linux-arm64-${VERSION}`
    }
    if (platform === 'windows' && arch === 'x64') {
        return `main-windows-amd64-${VERSION}`
    }
    if (platform === 'windows' && arch === 'arm64') {
        return `main-windows-arm64-${VERSION}`
    }

    console.error(`Unsupported platform (${platform}) and architecture (${arch})`)
    process.exit(1)
}

function main() {
    const binary = chooseBinary()
    const mainScript = `${__dirname}/${binary}`
    const spawnSyncReturns = childProcess.spawnSync(mainScript, { stdio: 'inherit' })
    const status = spawnSyncReturns.status
    if (typeof status === 'number') {
        process.exit(status)
    }
    process.exit(1)
}

if (require.main === module) {
    main()
}