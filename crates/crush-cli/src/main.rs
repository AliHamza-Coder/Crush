mod cli;
mod tui;

use clap::Parser;
use cli::Args;

fn main() -> anyhow::Result<()> {
    crush_core::core::deps::prepare_ort();

    let args = Args::parse();

    if args.version_flag {
        println!("CRUSH {}", crush_core::VERSION);
        return Ok(());
    }

    match args.command {
        Some(cli::Command::Analyse { dir, json }) => {
            cli::run_analyse(&dir, json)?;
        }
        Some(cli::Command::Install) => {
            cli::run_install()?;
        }
        Some(cli::Command::Uninstall) => {
            cli::run_uninstall()?;
        }
        Some(cli::Command::Update) => {
            cli::run_update()?;
        }
        Some(cli::Command::CheckDeps) => {
            cli::run_check_deps()?;
        }
        Some(cli::Command::Setup) => {
            cli::run_setup()?;
        }
        Some(cli::Command::AiCheck) => {
            cli::run_ai_check()?;
        }
        Some(cli::Command::Version) => {
            println!("CRUSH {}", crush_core::VERSION);
        }
        None => {
            if args.input.is_some() || args.format.is_some() {
                cli::run_direct(args)?;
            } else {
                tui::run()?;
            }
        }
    }

    Ok(())
}
