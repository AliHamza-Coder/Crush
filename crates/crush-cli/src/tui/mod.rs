pub mod app;
pub mod ui;

use crossterm::{
    event::{self, Event, KeyCode, KeyEventKind, MouseEvent, MouseEventKind, MouseButton},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use ratatui::prelude::*;

use self::app::{App, Panel};

pub fn run() -> anyhow::Result<()> {
    // The TUI event loop is synchronous, but processing is spawned with
    // `tokio::spawn`. Keep a multi-thread runtime alive (and entered) for the
    // whole session so spawned tasks actually run and `Handle::current()`
    // works inside `App::tick` / `App::start_processing`.
    let runtime = tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()
        .map_err(|e| anyhow::anyhow!("Failed to start async runtime: {e}"))?;
    let _guard = runtime.enter();

    enable_raw_mode()?;
    let mut stdout = std::io::stdout();
    execute!(stdout, EnterAlternateScreen, crossterm::event::EnableMouseCapture)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let mut app = App::new();
    app.init()?;

    let result = run_app(&mut terminal, &mut app);

    disable_raw_mode()?;
    execute!(
        terminal.backend_mut(),
        LeaveAlternateScreen,
        crossterm::event::DisableMouseCapture
    )?;
    terminal.show_cursor()?;

    result
}

fn run_app(terminal: &mut Terminal<CrosstermBackend<std::io::Stdout>>, app: &mut App) -> anyhow::Result<()> {
    loop {
        terminal.draw(|f| ui::render(f, app))?;

        if event::poll(std::time::Duration::from_millis(100))? {
            match event::read()? {
                Event::Key(key) => {
                    if key.kind == KeyEventKind::Press {
                        handle_key(app, key.code)?;
                    }
                }
                Event::Mouse(mouse) => {
                    handle_mouse(app, mouse);
                }
                _ => {}
            }
        }

        app.tick();

        if app.should_quit {
            return Ok(());
        }
    }
}

fn handle_key(app: &mut App, code: KeyCode) -> anyhow::Result<()> {
    if app.input_mode {
        match code {
            KeyCode::Enter => {
                let input = app.input_buffer.trim().to_string();
                app.input_mode = false;
                if !input.is_empty() {
                    if app.input_prompt.contains("directory") {
                        let path = std::path::PathBuf::from(&input);
                        if path.is_dir() {
                            app.dir = path;
                            app.scan()?;
                            app.status_msg = format!("Directory: {}", app.dir.display());
                        } else {
                            app.status_msg = "Invalid directory".into();
                        }
                    } else if app.input_prompt.contains("quality") {
                        if let Ok(q) = input.parse::<u8>() {
                            app.set_custom_quality(q);
                            app.status_msg = format!("Quality set to {}%", q);
                            app.active_panel = Panel::Menu;
                        } else {
                            app.status_msg = "Invalid quality (1-100)".into();
                        }
                    } else if app.input_prompt.contains("range") {
                        app.parse_range_selection(&input)?;
                        if !app.selected_indices.is_empty() {
                            app.filter_type = "custom".into();
                            app.start_processing()?;
                        }
                        app.active_panel = Panel::Menu;
                    }
                }
            }
            KeyCode::Esc => {
                app.input_mode = false;
                app.input_buffer.clear();
            }
            KeyCode::Char(c) => {
                app.input_buffer.push(c);
            }
            KeyCode::Backspace => {
                app.input_buffer.pop();
            }
            _ => {}
        }
        return Ok(());
    }

    match app.active_panel {
        Panel::Menu => match code {
            KeyCode::Char('q') | KeyCode::Esc => app.should_quit = true,
            KeyCode::Up | KeyCode::Char('k') => app.prev_menu(),
            KeyCode::Down | KeyCode::Char('j') => app.next_menu(),
            KeyCode::Enter => app.execute_menu_action()?,
            KeyCode::Char('h') | KeyCode::Char('H') => {
                app.active_panel = Panel::Help;
            }
            KeyCode::Char('1') => { app.menu_idx = 0; app.execute_menu_action()?; }
            KeyCode::Char('2') => { app.menu_idx = 1; app.execute_menu_action()?; }
            KeyCode::Char('3') => { app.menu_idx = 2; app.execute_menu_action()?; }
            KeyCode::Char('4') => { app.menu_idx = 3; app.execute_menu_action()?; }
            KeyCode::Char('5') => { app.menu_idx = 4; app.execute_menu_action()?; }
            KeyCode::Char('6') => { app.menu_idx = 5; app.execute_menu_action()?; }
            KeyCode::Char('7') => { app.menu_idx = 6; app.execute_menu_action()?; }
            KeyCode::Char('8') => { app.menu_idx = 7; app.execute_menu_action()?; }
            KeyCode::Char('9') => { app.menu_idx = 8; app.execute_menu_action()?; }
            _ => {}
        },
        Panel::Files => match code {
            KeyCode::Char('q') | KeyCode::Esc => {
                app.active_panel = Panel::Menu;
                app.select_mode = false;
            }
            KeyCode::Up | KeyCode::Char('k') => app.prev_file(),
            KeyCode::Down | KeyCode::Char('j') => app.next_file(),
            KeyCode::Char(' ') => {
                if app.select_mode {
                    app.toggle_select_file();
                }
            }
            KeyCode::Enter => {
                if app.select_mode {
                    app.confirm_select_files()?;
                } else {
                    app.active_panel = Panel::Menu;
                }
            }
            KeyCode::Tab => {
                app.active_panel = Panel::Menu;
                app.select_mode = false;
            }
            _ => {}
        },
        Panel::Quality => match code {
            KeyCode::Char('q') | KeyCode::Esc | KeyCode::Tab => {
                app.active_panel = Panel::Menu;
            }
            KeyCode::Up | KeyCode::Char('k') => app.prev_quality(),
            KeyCode::Down | KeyCode::Char('j') => app.next_quality(),
            KeyCode::Enter => app.active_panel = Panel::Menu,
            KeyCode::Char('l') | KeyCode::Char('L') => {
                app.lossless = !app.lossless;
                app.quality = if app.lossless { 0 } else { 85 };
                app.status_msg = if app.lossless { "Lossless ON".into() } else { "Lossless OFF".into() };
            }
            KeyCode::Char(c) if c.is_ascii_digit() => {
                app.input_mode = true;
                app.input_buffer.clear();
                app.input_buffer.push(c);
                app.input_prompt = "Enter quality (1-100):".into();
                app.status_msg = "Type quality number, press Enter".into();
            }
            _ => {}
        },
        Panel::Format => match code {
            KeyCode::Char('q') | KeyCode::Esc | KeyCode::Tab => {
                app.active_panel = Panel::Menu;
            }
            KeyCode::Up | KeyCode::Char('k') => app.prev_format(),
            KeyCode::Down | KeyCode::Char('j') => app.next_format(),
            KeyCode::Enter => {
                if matches!(app.mode, app::ProcessMode::Convert) {
                    app.submenu_flow = app::SubmenuFlow::FormatFirst;
                    app.start_submenu_flow(app::SubmenuFlow::FormatFirst);
                } else {
                    app.active_panel = Panel::Menu;
                }
            }
            _ => {}
        },
        Panel::QualitySubmenu => match code {
            KeyCode::Char('q') | KeyCode::Esc | KeyCode::Tab => {
                app.active_panel = Panel::Menu;
                app.submenu_flow = app::SubmenuFlow::None;
            }
            KeyCode::Up | KeyCode::Char('k') => {
                if app.submenu_idx > 0 {
                    app.submenu_idx -= 1;
                } else {
                    app.submenu_idx = app.submenu_items.len() - 1;
                }
            }
            KeyCode::Down | KeyCode::Char('j') => {
                if app.submenu_idx < app.submenu_items.len() - 1 {
                    app.submenu_idx += 1;
                } else {
                    app.submenu_idx = 0;
                }
            }
            KeyCode::Enter => {
                app.confirm_submenu_quality();
            }
            KeyCode::Char('l') | KeyCode::Char('L') => {
                app.lossless = !app.lossless;
                app.quality = if app.lossless { 0 } else { 85 };
                app.status_msg = if app.lossless { "Lossless ON".into() } else { "Lossless OFF".into() };
            }
            _ => {}
        },
        Panel::BackupConfirm => match code {
            KeyCode::Char('q') | KeyCode::Esc | KeyCode::Tab => {
                app.active_panel = Panel::Menu;
                app.submenu_flow = app::SubmenuFlow::None;
            }
            KeyCode::Up | KeyCode::Char('k') => {
                app.submenu_idx = if app.submenu_idx > 0 { app.submenu_idx - 1 } else { app.submenu_items.len() - 1 };
            }
            KeyCode::Down | KeyCode::Char('j') => {
                app.submenu_idx = if app.submenu_idx < app.submenu_items.len() - 1 { app.submenu_idx + 1 } else { 0 };
            }
            KeyCode::Enter => {
                app.confirm_backup();
            }
            _ => {}
        },
        Panel::Queue => match code {
            KeyCode::Char('q') | KeyCode::Esc | KeyCode::Tab => {
                if !app.processing {
                    app.active_panel = Panel::Menu;
                }
            }
            KeyCode::Char('c') => app.cancel(),
            KeyCode::Char('x') => app.clear_queue(),
            _ => {}
        },
        Panel::Help => match code {
            KeyCode::Char('h') | KeyCode::Char('H') | KeyCode::Esc | KeyCode::Tab => {
                app.active_panel = Panel::Menu;
            }
            _ => {}
        },
    }
    Ok(())
}

fn handle_mouse(app: &mut App, mouse: MouseEvent) {
    if app.input_mode {
        return;
    }

    match mouse.kind {
        MouseEventKind::Down(MouseButton::Left) => {
            let col = mouse.column as usize;
            let row = mouse.row as usize;
            let item_y = app.geom.item_y as usize;

            match app.active_panel {
                Panel::Menu => {
                    let menu_x = app.geom.menu_x as usize;
                    let menu_w = app.geom.menu_w as usize;
                    if col >= menu_x && col < menu_x + menu_w && row >= item_y {
                        let idx = row - item_y;
                        let menu_len = App::menu_items().len();
                        if idx < menu_len {
                            app.menu_idx = idx;
                            let _ = app.execute_menu_action();
                        }
                    }
                }
                Panel::Files => {
                    if row >= item_y {
                        let idx = row - item_y;
                        let files = app.get_filtered_files();
                        if idx < files.len() {
                            app.selected_file = idx;
                            if app.select_mode {
                                app.toggle_select_file();
                            }
                        }
                    }
                }
                Panel::Quality => {
                    if row >= item_y {
                        let idx = row - item_y;
                        if idx < 6 {
                            app.quality_idx = idx;
                            app.quality = match idx {
                                0 => 100,
                                1 => 90,
                                2 => 85,
                                3 => 75,
                                4 => 60,
                                _ => 85,
                            };
                        }
                    }
                }
                Panel::Format => {
                    if row >= item_y {
                        let idx = row - item_y;
                        let formats = app.available_formats();
                        if idx < formats.len() {
                            app.format_idx = idx;
                            app.format = formats[idx].clone();
                        }
                    }
                }
                Panel::Queue => {}
                Panel::QualitySubmenu | Panel::BackupConfirm => {
                    if row >= item_y {
                        let idx = row - item_y;
                        if idx < app.submenu_items.len() {
                            app.submenu_idx = idx;
                        }
                    }
                }
                Panel::Help => {
                    app.active_panel = Panel::Menu;
                }
            }
        }
        MouseEventKind::ScrollUp => {
            match app.active_panel {
                Panel::Menu => app.prev_menu(),
                Panel::Files => app.prev_file(),
                Panel::Quality => app.prev_quality(),
                Panel::Format => app.prev_format(),
                Panel::QualitySubmenu => {
                    if app.submenu_idx > 0 {
                        app.submenu_idx -= 1;
                    } else {
                        app.submenu_idx = app.submenu_items.len() - 1;
                    }
                }
                Panel::BackupConfirm => {
                    app.submenu_idx = if app.submenu_idx > 0 { app.submenu_idx - 1 } else { app.submenu_items.len() - 1 };
                }
                _ => {}
            }
        }
        MouseEventKind::ScrollDown => {
            match app.active_panel {
                Panel::Menu => app.next_menu(),
                Panel::Files => app.next_file(),
                Panel::Quality => app.next_quality(),
                Panel::Format => app.next_format(),
                Panel::QualitySubmenu => {
                    if app.submenu_idx < app.submenu_items.len() - 1 {
                        app.submenu_idx += 1;
                    } else {
                        app.submenu_idx = 0;
                    }
                }
                Panel::BackupConfirm => {
                    app.submenu_idx = if app.submenu_idx < app.submenu_items.len() - 1 { app.submenu_idx + 1 } else { 0 };
                }
                _ => {}
            }
        }
        _ => {}
    }
}
