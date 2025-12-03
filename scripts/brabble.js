#!/usr/bin/env node
import { execa } from 'execa';
import chalk from 'chalk';
import { existsSync } from 'fs';
import { join } from 'path';

const BIN = join(process.cwd(), 'bin', 'brabble');

function printHelp() {
  console.log(chalk.bold.blue('Brabble (pnpm helper)'));
  console.log(chalk.dim('Builds the Go binary if needed, then runs it.'));
  console.log('');
  console.log(chalk.bold('Usage'));
  console.log('  pnpm brabble             ', chalk.dim('build + serve in foreground'));
  console.log('  pnpm brabble <args...>   ', chalk.dim('build + run ./bin/brabble <args>'));
  console.log('  pnpm brabble --help      ', chalk.dim('show this help'));
  console.log('  pnpm brabble --version   ', chalk.dim('show brabble version'));
  console.log('');
  console.log(chalk.bold('Key commands'));
  console.log('  start | stop | restart          ', chalk.dim('daemon lifecycle'));
  console.log('  status --json                   ', chalk.dim('uptime + last transcripts'));
  console.log('  list-mics | set-mic "<name>"    ', chalk.dim('select input device (whisper build)'));
  console.log('  doctor                          ', chalk.dim('check model/hook/portaudio'));
  console.log('  setup                           ', chalk.dim('download default whisper model'));
  console.log('  models list|download|set        ', chalk.dim('manage whisper.cpp models'));
  console.log('  install-service --env KEY=VAL   ', chalk.dim('write launchd plist (macOS)'));
  console.log('  reload                          ', chalk.dim('reload hook/wake config live'));
  console.log('  health                          ', chalk.dim('control-socket liveness ping'));
  console.log('');
  console.log(chalk.bold('Examples'));
  console.log('  pnpm brabble start --metrics-addr 127.0.0.1:9317');
  console.log('  pnpm brabble list-mics');
  console.log('  pnpm brabble models download ggml-medium-q5_1.bin');
  console.log('  pnpm brabble models set ggml-medium-q5_1.bin');
  console.log('  pnpm brabble install-service --env BRABBLE_METRICS_ADDR=127.0.0.1:9317');
}

async function ensureBuilt() {
  if (existsSync(BIN)) return;
  console.log(chalk.yellow('Building brabble binary...'));
  await execa('go', ['build', '-o', BIN, './cmd/brabble'], { stdio: 'inherit' });
}

async function main() {
  const args = process.argv.slice(2);
  if (args.includes('--help') || args.includes('-h')) {
    printHelp();
    return;
  }
  if (args.includes('--version') || args.includes('-v')) {
    await ensureBuilt();
    const { stdout } = await execa(BIN, ['--version']);
    console.log(stdout.trim());
    return;
  }
  await ensureBuilt();
  const runArgs = args.length === 0 ? ['serve'] : args;
  const child = execa(BIN, runArgs, { stdio: 'inherit' });
  child.on('exit', code => process.exit(code ?? 0));
}

main().catch(err => {
  console.error(chalk.red('Error:'), err.message);
  process.exit(1);
});
