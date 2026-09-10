use ratatui::prelude::*;
use ratatui::widgets::*;
use crush_core::core::fileutil;

use super::app::{App, Panel};

pub fn render(f: &mut Frame, app: &mut App) {
    let area = f.area();
    let main_layout = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Length(3), Constraint::Min(0), Constraint::Length(3)])
        .split(area);

    render_header(f, main_layout[0], app);

    let body = main_layout[1];
    match app.active_panel {
        Panel::Menu => render_menu_view(f, body, app),
        Panel::Files => render_files_view(f, body, app),
        Panel::Quality => render_quality_view(f, body, app),
        Panel::Format => render_format_view(f, body, app),
        Panel::Queue => render_queue_view(f, body, app),
        Panel::QualitySubmenu => render_quality_submenu(f, body, app),
        Panel::BackupConfirm => render_backup_confirm(f, body, app),
        Panel::Help => render_help(f, body, app),
    }

    render_summary_bar(f, main_layout[2], app);
}

// ─── helpers ────────────────────────────────────────────────────────────────

fn trunc(s: &str, max: usize) -> String {
    let chars: Vec<char> = s.chars().take(max).collect();
    let total: usize = s.chars().count();
    if total <= max {
        chars.iter().collect()
    } else {
        let mut out: String = chars.iter().collect();
        out.push('…');
        out
    }
}

fn pct(p: f64, w: usize) -> String {
    let filled = (p * w as f64).round() as usize;
    let filled = filled.min(w);
    format!("{}{}", "█".repeat(filled), "░".repeat(w - filled))
}

fn chip(label: &str, color: Color) -> Span<'static> {
    Span::styled(
        format!(" {} ", label),
        Style::default().fg(color),
    )
}

// ─── header ─────────────────────────────────────────────────────────────────

fn render_header(f: &mut Frame, area: Rect, app: &App) {
    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Cyan));

    let w = area.width as usize;
    let title = format!("CRUSH {}", crush_core::VERSION);
    let center = "Multimedia Mission Control";

    let ff_pill = if app.ffmpeg.is_some() { "FFmpeg ●" } else { "FFmpeg ○" };
    let onnx_pill = if app.ai.is_model_available() { "ONNX ●" } else { "ONNX ○" };
    let right_str = format!("{}  Native ●  {} ", ff_pill, onnx_pill);
    let right_len: usize = right_str.chars().count();

    let title_len = title.len();
    let center_len = center.len();
    let avail = w.saturating_sub(2);
    let inner = avail.saturating_sub(title_len).saturating_sub(right_len);
    let pad = if inner > center_len { (inner - center_len) / 2 } else { 0 };
    let content = format!(
        "{}{}{}{}{}",
        title,
        " ".repeat(pad),
        center,
        " ".repeat(avail.saturating_sub(title_len + pad + center_len + right_len)),
        right_str
    );

    let line = Line::from(vec![
        Span::styled("◆ ", Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD)),
        Span::styled(
            content,
            Style::default().fg(Color::White),
        ),
    ]);

    f.render_widget(Paragraph::new(line).block(block), area);
}

// ─── menu view ──────────────────────────────────────────────────────────────

fn render_menu_view(f: &mut Frame, area: Rect, app: &mut App) {
    let menu_w: u16 = (area.width as f64 * 0.34).round() as u16;
    let menu_w = menu_w.clamp(36, 52);

    let chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Min(0), Constraint::Length(menu_w)])
        .split(area);

    render_analysis(f, chunks[0], app);

    let menu_area = chunks[1];
    app.geom.menu_x = menu_area.x;
    app.geom.menu_w = menu_area.width;
    app.geom.item_y = menu_area.y + 1;

    render_menu(f, menu_area, app);
}

// ─── analysis dashboard ─────────────────────────────────────────────────────

fn render_analysis(f: &mut Frame, area: Rect, app: &App) {
    let short_name = app.dir.file_name()
        .map(|n| n.to_string_lossy().into_owned())
        .unwrap_or_else(|| app.dir.display().to_string());

    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Cyan))
        .title(Span::styled(
            format!(" {} ", short_name),
            Style::default().add_modifier(Modifier::BOLD).fg(Color::Cyan),
        ))
        .title_alignment(Alignment::Center);

    if let Some(ref stats) = app.stats {
        let w = area.width.saturating_sub(4) as usize;
        let all_count = (stats.images + stats.videos + stats.audio).max(1);
        let all_size = stats.total_size.max(1);

        let mut lines: Vec<Line> = Vec::new();

        // ── category bars ──
        let cats: &[(usize, u64, Color, &str)] = &[
            (stats.images, stats.image_size, Color::Green, "Images"),
            (stats.videos, stats.video_size, Color::Cyan, "Videos"),
            (stats.audio, stats.audio_size, Color::Yellow, "Audio"),
        ];

        for &(count, size, color, label) in cats {
            let p = count as f64 / all_count as f64;
            let bar_w = 20.min(w.saturating_sub(26));
            lines.push(Line::from(vec![
                Span::styled(format!("  {:<7}", label), Style::default().fg(color).add_modifier(Modifier::BOLD)),
                Span::styled(pct(p, bar_w), Style::default().fg(color)),
                Span::styled(format!(" {:>3} ", count), Style::default().fg(Color::White)),
                Span::styled(fileutil::format_size(size), Style::default().fg(Color::DarkGray)),
            ]));
        }

        lines.push(Line::from(""));

        // ── total line ──
        lines.push(Line::from(vec![
            Span::styled("  Total ", Style::default().add_modifier(Modifier::BOLD).fg(Color::White)),
            Span::styled(
                format!("{} files  │  {}", stats.total, fileutil::format_size(stats.total_size)),
                Style::default().fg(Color::DarkGray),
            ),
        ]));

        // ── storage share bar ──
        if stats.total_size > 0 {
            let share_w = 30.min(w.saturating_sub(10));
            let si = (stats.image_size as f64 / all_size as f64 * share_w as f64).round() as usize;
            let sv = (stats.video_size as f64 / all_size as f64 * share_w as f64).round() as usize;
            let sa = share_w.saturating_sub(si + sv);
            let bar = format!(
                "{}{}{}",
                "█".repeat(si.min(share_w)),
                "█".repeat(sv.min(share_w.saturating_sub(si))),
                "█".repeat(sa),
            );
            let colors: Vec<Color> = std::iter::repeat(Color::Green)
                .take(si)
                .chain(std::iter::repeat(Color::Cyan).take(sv))
                .chain(std::iter::repeat(Color::Yellow).take(sa))
                .collect();
            let spans: Vec<Span> = bar.chars().zip(colors.into_iter()).map(|(c, cl)| {
                Span::styled(c.to_string(), Style::default().fg(cl))
            }).collect();
            lines.push(Line::from(vec![
                Span::styled("  Share  ", Style::default().fg(Color::DarkGray)),
            ]));
            lines.push(Line::from(spans));
        }

        lines.push(Line::from(""));

        // ── format chips ──
        let format_order = [
            ".jpg", ".jpeg", ".png", ".webp", ".avif", ".gif", ".svg",
            ".mp4", ".mov", ".webm", ".mkv", ".avi",
            ".mp3", ".wav", ".flac", ".ogg", ".m4a", ".aac", ".opus", ".alac",
        ];

        let mut chip_spans: Vec<Span> = Vec::new();
        chip_spans.push(Span::styled("  Formats  ", Style::default().add_modifier(Modifier::BOLD).fg(Color::DarkGray)));
        for ext in &format_order {
            if let Some(&count) = stats.formats.get(*ext) {
                let color = match fileutil::detect_type(ext) {
                    fileutil::FileType::Image => Color::Green,
                    fileutil::FileType::Video => Color::Cyan,
                    _ => Color::Yellow,
                };
                let label = format!("{}×{}", ext.trim_start_matches('.').to_uppercase(), count);
                chip_spans.push(chip(&label, color));
            }
        }
        if chip_spans.len() > 1 {
            lines.push(Line::from(chip_spans));
        }

        if stats.total == 0 {
            lines.push(Line::from(""));
            lines.push(Line::from(vec![
                Span::styled("  No media files in this directory", Style::default().fg(Color::Yellow)),
            ]));
            lines.push(Line::from(vec![
                Span::styled("  Click \"Change dir…\" to browse", Style::default().fg(Color::DarkGray)),
            ]));
        }

        let list = List::new(
            lines.into_iter().map(ListItem::new).collect::<Vec<_>>(),
        ).block(block);
        f.render_widget(list, area);
    } else {
        f.render_widget(Paragraph::new("No directory scanned").block(block), area);
    }
}

// ─── menu list ──────────────────────────────────────────────────────────────

fn render_menu(f: &mut Frame, area: Rect, app: &App) {
    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Cyan))
        .title(Span::styled(
            " ⚡ Actions ",
            Style::default().add_modifier(Modifier::BOLD).fg(Color::Cyan),
        ))
        .title_alignment(Alignment::Center);

    let menu_items = App::menu_items();
    let items: Vec<ListItem> = menu_items.iter().enumerate().map(|(i, action)| {
        let sel = i == app.menu_idx;
        let style = if sel {
            Style::default().fg(Color::Black).bg(Color::Cyan).add_modifier(Modifier::BOLD)
        } else {
            Style::default()
        };

        let (icon, color) = match action {
            super::app::MenuAction::CompressAll => ("◆", Color::Green),
            super::app::MenuAction::CompressImages => ("🖼", Color::Green),
            super::app::MenuAction::CompressVideos => ("🎬", Color::Green),
            super::app::MenuAction::CompressAudio => ("🎵", Color::Green),
            super::app::MenuAction::ConvertAll => ("◇", Color::Cyan),
            super::app::MenuAction::ConvertImages => ("🖼", Color::Cyan),
            super::app::MenuAction::ConvertVideos => ("🎬", Color::Cyan),
            super::app::MenuAction::ConvertAudio => ("🎵", Color::Cyan),
            super::app::MenuAction::ExtractAudio => ("🔊", Color::Yellow),
            super::app::MenuAction::SelectFiles => ("#", Color::White),
            super::app::MenuAction::Arrange => ("📂", Color::Magenta),
            super::app::MenuAction::ChangeDir => ("📁", Color::Magenta),
            super::app::MenuAction::Favicon => ("🌐", Color::Yellow),
            super::app::MenuAction::CheckDeps => ("🔍", Color::Cyan),
            super::app::MenuAction::Quit => ("✕", Color::Red),
        };

        let prefix = if sel { "▸" } else { " " };
        ListItem::new(Line::from(vec![
            Span::styled(format!(" {} ", prefix), if sel { style } else { Style::default().fg(Color::DarkGray) }),
            Span::styled(format!("{} ", icon), if sel { style } else { Style::default().fg(color) }),
            Span::styled(action.label(), style),
        ]))
    }).collect();

    let list = List::new(items).block(block);
    f.render_stateful_widget(list, area, &mut ListState::default().with_selected(Some(app.menu_idx)));
}

// ─── files view ─────────────────────────────────────────────────────────────

fn render_files_view(f: &mut Frame, area: Rect, app: &mut App) {
    let files = app.get_filtered_files();
    let title = if app.select_mode {
        format!(" 📋 Select Files ({} of {} selected) ", app.selected_indices.len(), files.len())
    } else {
        format!(" 📋 Files ({}) ", files.len())
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(if app.select_mode { Color::Yellow } else { Color::Cyan }))
        .title(Span::styled(title, Style::default().add_modifier(Modifier::BOLD).fg(if app.select_mode { Color::Yellow } else { Color::Cyan })))
        .title_alignment(Alignment::Center);

    if files.is_empty() {
        f.render_widget(Paragraph::new("No files in this directory").block(block), area);
        return;
    }

    let inner = block.inner(area);
    let item_start_y = inner.y + 2;

    app.geom.item_y = item_start_y;
    app.geom.menu_x = 0;
    app.geom.menu_w = inner.width;

    let header_line = Line::from(vec![
        Span::styled(
            "  #    Type      Size       Format   Name",
            Style::default().fg(Color::DarkGray).add_modifier(Modifier::BOLD),
        ),
    ]);

    let mut items = vec![
        ListItem::new(header_line),
        ListItem::new(Line::from("")),
    ];

    let name_w = (inner.width as usize).saturating_sub(34).max(8);

    for (i, file) in files.iter().enumerate() {
        let (icon, color) = match file.file_type {
            fileutil::FileType::Image => ("🖼", Color::Green),
            fileutil::FileType::Video => ("🎬", Color::Cyan),
            fileutil::FileType::Audio => ("🎵", Color::Yellow),
            _ => ("📄", Color::DarkGray),
        };

        let is_cursor = i == app.selected_file;
        let is_checked = app.select_mode && app.selected_indices.contains(&file.index);

        let style = if is_cursor {
            if is_checked {
                Style::default().fg(Color::Black).bg(Color::Green).add_modifier(Modifier::BOLD)
            } else {
                Style::default().fg(Color::Black).bg(Color::Cyan).add_modifier(Modifier::BOLD)
            }
        } else if is_checked {
            Style::default().fg(Color::Green).add_modifier(Modifier::BOLD)
        } else {
            Style::default().fg(color)
        };

        let check = if app.select_mode {
            if is_checked { " ✓" } else { "  " }
        } else {
            ""
        };

        items.push(ListItem::new(Line::from(vec![
            Span::styled(format!("{} {:<3} ", check, file.index), style),
            Span::styled(icon, Style::default().fg(color)),
            Span::styled(format!(" {:<8} ", file.type_name), style),
            Span::styled(format!("{:<10} ", file.size_str), style),
            Span::styled(format!("{:<6} ", file.ext.trim_start_matches('.').to_uppercase()), style),
            Span::styled(trunc(&file.name, name_w), style),
        ])));
    }

    let list = List::new(items).block(block);
    f.render_stateful_widget(list, area, &mut ListState::default().with_selected(Some(app.selected_file)));
}

// ─── quality (full panel) ───────────────────────────────────────────────────

fn render_quality_view(f: &mut Frame, area: Rect, app: &App) {
    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Cyan))
        .title(Span::styled(
            " 🎚 Quality Selection ",
            Style::default().add_modifier(Modifier::BOLD).fg(Color::Cyan),
        ))
        .title_alignment(Alignment::Center);

    let qualities = [
        (100, "Maximum", "best quality, largest file", Color::Green),
        (90,  "High",    "slightly larger", Color::Cyan),
        (85,  "Balanced ★", "good quality, ~50-70% smaller", Color::Green),
        (75,  "Smaller", "slightly lower quality", Color::Yellow),
        (60,  "Compact", "good for web sharing", Color::Yellow),
        (0,   "Lossless", "original quality preserved", Color::Magenta),
    ];

    let mut list_items: Vec<ListItem> = qualities.iter().enumerate().map(|(i, &(q, label, desc, color))| {
        let sel = i == app.quality_idx;
        let style = if sel {
            Style::default().fg(Color::Black).bg(Color::Cyan).add_modifier(Modifier::BOLD)
        } else {
            Style::default()
        };
        let q_str = if q > 0 { format!("{:>3}", q) } else { "  L".into() };
        ListItem::new(Line::from(vec![
            Span::styled(format!("  {} ", q_str), Style::default().fg(color).add_modifier(if sel { Modifier::BOLD } else { Modifier::empty() })),
            Span::styled(format!("{:<12} ", label), style),
            Span::styled(desc, Style::default().fg(Color::DarkGray)),
        ]))
    }).collect();

    if app.lossless {
        list_items.push(ListItem::new(Line::from("")));
        list_items.push(ListItem::new(Line::from(vec![
            Span::styled("  ⚡ Lossless mode ON", Style::default().fg(Color::Magenta).add_modifier(Modifier::BOLD)),
        ])));
    }

    let list = List::new(list_items).block(block);
    f.render_stateful_widget(list, area, &mut ListState::default().with_selected(Some(app.quality_idx)));
}

// ─── format (full panel) ────────────────────────────────────────────────────

fn render_format_view(f: &mut Frame, area: Rect, app: &App) {
    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Cyan))
        .title(Span::styled(
            " 🎯 Target Format ",
            Style::default().add_modifier(Modifier::BOLD).fg(Color::Cyan),
        ))
        .title_alignment(Alignment::Center);

    let formats = app.available_formats();
    let items: Vec<ListItem> = formats.iter().enumerate().map(|(i, fmt)| {
        let sel = i == app.format_idx;
        let (color, cat) = match fmt.as_str() {
            "webp" | "avif" | "png" | "jpg" | "bmp" => (Color::Green, "IMG"),
            "mp4" | "webm" | "mkv" | "mov" | "avi" => (Color::Cyan, "VID"),
            "mp3" | "flac" | "ogg" | "wav" | "aac" | "opus" | "m4a" | "alac" => (Color::Yellow, "AUD"),
            _ => (Color::White, "???"),
        };
        let style = if sel {
            Style::default().fg(Color::Black).bg(Color::Cyan).add_modifier(Modifier::BOLD)
        } else {
            Style::default().fg(color)
        };
        ListItem::new(Line::from(vec![
            Span::styled(format!("  [{:<3}] ", cat), Style::default().fg(Color::DarkGray)),
            Span::styled(fmt.to_uppercase(), style),
        ]))
    }).collect();

    let list = List::new(items).block(block);
    f.render_stateful_widget(list, area, &mut ListState::default().with_selected(Some(app.format_idx)));
}

// ─── queue ──────────────────────────────────────────────────────────────────

fn render_queue_view(f: &mut Frame, area: Rect, app: &App) {
    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Cyan))
        .title(Span::styled(
            " ⚡ Processing Queue ",
            Style::default().add_modifier(Modifier::BOLD).fg(Color::Cyan),
        ))
        .title_alignment(Alignment::Center);

    let state = match app.queue_state.as_ref() {
        Some(s) if !s.tasks.is_empty() => s,
        _ => {
            let msg = if app.processing {
                "Processing…"
            } else {
                "Press Enter on a menu action to start"
            };
            f.render_widget(Paragraph::new(msg).block(block), area);
            return;
        }
    };

    let inner = block.inner(area);

    // ── overall gauge ──
    let done = state.completed + state.failed;
    let pct_val = if state.total > 0 { (done as f64 / state.total as f64 * 100.0) as u16 } else { 0 };

    let gauge = Gauge::default()
        .block(Block::default().borders(Borders::BOTTOM).border_style(Style::default().fg(Color::Cyan)))
        .gauge_style(Style::default().fg(Color::Cyan).bg(Color::Black))
        .ratio(done as f64 / state.total as f64)
        .label(Span::styled(
            format!("{}%  {}/{}", pct_val, done, state.total),
            Style::default().fg(Color::White).add_modifier(Modifier::BOLD),
        ));
    f.render_widget(gauge, inner);

    // ── task rows ──
    let row_area = Rect {
        x: inner.x,
        y: inner.y + 2,
        width: inner.width,
        height: inner.height.saturating_sub(2),
    };

    let rows: Vec<ListItem> = state.tasks.iter().map(|task| {
        let engine_color = match task.engine {
            crush_core::core::queue::Engine::Ffmpeg => Color::Cyan,
            crush_core::core::queue::Engine::Native => Color::Green,
            crush_core::core::queue::Engine::OnnxAi => Color::Magenta,
        };
        let engine_name = match task.engine {
            crush_core::core::queue::Engine::Ffmpeg => "FFM",
            crush_core::core::queue::Engine::Native => "NAT",
            crush_core::core::queue::Engine::OnnxAi => "AI",
        };

        let (icon, icon_c) = match task.status {
            crush_core::core::queue::TaskStatus::Completed => ("✓", Color::Green),
            crush_core::core::queue::TaskStatus::Failed    => ("✗", Color::Red),
            crush_core::core::queue::TaskStatus::Running   => ("▶", Color::Cyan),
            crush_core::core::queue::TaskStatus::Pending   => ("○", Color::DarkGray),
            crush_core::core::queue::TaskStatus::Skipped   => ("⏭", Color::Yellow),
        };

        let bar_w = 16.min(row_area.width.saturating_sub(40) as usize);
        let bar = pct(task.progress as f64 / 100.0, bar_w);

        let err = task.error.as_ref()
            .map(|e| Span::styled(format!("  {}", trunc(e, 24)), Style::default().fg(Color::Red)))
            .unwrap_or_default();

        ListItem::new(Line::from(vec![
            Span::styled(format!(" {} ", icon), Style::default().fg(icon_c)),
            Span::styled(format!("{:<16} ", trunc(&task.file_name, 16)), Style::default().fg(Color::White)),
            Span::styled(bar, Style::default().fg(engine_color)),
            Span::styled(format!(" {:>5.0}% ", task.progress), Style::default().fg(Color::DarkGray)),
            Span::styled(format!("{:<4} ", engine_name), Style::default().fg(engine_color)),
            err,
        ]))
    }).collect();

    let list = List::new(rows);
    f.render_widget(list, row_area);
}

// ─── quality submenu (centered card) ────────────────────────────────────────

fn render_quality_submenu(f: &mut Frame, area: Rect, app: &mut App) {
    let card_w: u16 = area.width.min(64);
    let card_h: u16 = (app.submenu_items.len() as u16 + 4).min(area.height);
    let cx = area.x + (area.width.saturating_sub(card_w)) / 2;
    let cy = area.y + (area.height.saturating_sub(card_h)) / 2;
    let card = Rect::new(cx, cy, card_w, card_h);

    app.geom.item_y = card.y + 1;

    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Cyan))
        .title(Span::styled(
            format!(" 🎚 Quality — {} files ", app.get_filtered_files().len()),
            Style::default().add_modifier(Modifier::BOLD).fg(Color::Cyan),
        ))
        .title_alignment(Alignment::Center);

    let items: Vec<ListItem> = app.submenu_items.iter().enumerate().map(|(i, item)| {
        let sel = i == app.submenu_idx;
        let style = if sel {
            Style::default().fg(Color::Black).bg(Color::Cyan).add_modifier(Modifier::BOLD)
        } else {
            Style::default()
        };

        let (color, prefix) = if item.contains("Lossless") {
            (Color::Magenta, "  L ")
        } else if item.contains("Custom") {
            (Color::White, "  ? ")
        } else if item.contains("★") {
            (Color::Green, "    ")
        } else {
            (Color::DarkGray, "    ")
        };

        ListItem::new(Line::from(vec![
            Span::styled(prefix, Style::default().fg(color)),
            Span::styled(item.clone(), style),
        ]))
    }).collect();

    let list = List::new(items).block(block);
    f.render_stateful_widget(list, card, &mut ListState::default().with_selected(Some(app.submenu_idx)));
}

// ─── backup confirm (centered card) ─────────────────────────────────────────

fn render_backup_confirm(f: &mut Frame, area: Rect, app: &mut App) {
    let card_w: u16 = area.width.min(50);
    let card_h: u16 = (app.submenu_items.len() as u16 + 4).min(area.height);
    let cx = area.x + (area.width.saturating_sub(card_w)) / 2;
    let cy = area.y + (area.height.saturating_sub(card_h)) / 2;
    let card = Rect::new(cx, cy, card_w, card_h);

    app.geom.item_y = card.y + 1;

    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Yellow))
        .title(Span::styled(
            " 📦 Backup Originals? ",
            Style::default().add_modifier(Modifier::BOLD).fg(Color::Yellow),
        ))
        .title_alignment(Alignment::Center);

    let items: Vec<ListItem> = app.submenu_items.iter().enumerate().map(|(i, item)| {
        let sel = i == app.submenu_idx;
        let style = if sel {
            Style::default().fg(Color::Black).bg(Color::Cyan).add_modifier(Modifier::BOLD)
        } else {
            Style::default()
        };
        let (icon, color) = if i == 0 { ("✓", Color::Green) } else { ("✕", Color::Red) };
        ListItem::new(Line::from(vec![
            Span::styled(format!("  {} ", icon), Style::default().fg(color)),
            Span::styled(item.clone(), style),
        ]))
    }).collect();

    let list = List::new(items).block(block);
    f.render_stateful_widget(list, card, &mut ListState::default().with_selected(Some(app.submenu_idx)));
}

// ─── help ───────────────────────────────────────────────────────────────────

fn render_help(f: &mut Frame, area: Rect, _app: &App) {
    let card_w: u16 = area.width.min(72);
    let card_h: u16 = area.height;
    let cx = area.x + (area.width.saturating_sub(card_w)) / 2;
    let card = Rect::new(cx, area.y, card_w, card_h);

    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Cyan))
        .title(Span::styled(
            " ❓ Help — Esc or H to close ",
            Style::default().add_modifier(Modifier::BOLD).fg(Color::Cyan),
        ))
        .title_alignment(Alignment::Center);

    let sections: Vec<(&str, Vec<(&str, &str)>, Color)> = vec![
        ("NAVIGATION", vec![
            ("↑ ↓ / j k", "Navigate items"),
            ("Enter", "Select / confirm"),
            ("Tab", "Switch panel / back"),
            ("Esc", "Back / quit"),
            ("q", "Quit (from menu)"),
            ("Mouse click", "Select item"),
            ("Scroll wheel", "Navigate up/down"),
        ], Color::Cyan),
        ("COMPRESS / CONVERT", vec![
            ("1-9", "Quick-select menu item"),
            ("L", "Toggle lossless"),
            ("0-9 (quality)", "Custom quality value"),
            ("Space", "Toggle file selection"),
        ], Color::Green),
        ("QUEUE", vec![
            ("c", "Cancel processing"),
            ("x", "Clear queue"),
        ], Color::Yellow),
        ("CLI", vec![
            ("crush", "Launch TUI"),
            ("crush analyse .", "Analyse directory"),
            ("crush -i <dir> -f <fmt>", "Direct compress/convert"),
            ("crush -i <dir> -f <fmt> --dry-run", "Preview"),
            ("crush install / uninstall", "Manage install"),
            ("crush version", "Print version"),
        ], Color::Magenta),
        ("FORMATS", vec![
            ("Images", "webp, avif, png, jpg, bmp"),
            ("Videos", "mp4, webm, mkv, mov, avi"),
            ("Audio", "mp3, flac, ogg, wav, aac, opus, m4a, alac"),
        ], Color::White),
        ("QUALITY PRESETS", vec![
            ("100%", "Maximum — best quality"),
            ("90%", "High — slightly larger"),
            ("85%", "Balanced ★ — ~50-70% smaller"),
            ("75%", "Smaller — slightly lower"),
            ("60%", "Compact — web sharing"),
            ("Lossless", "Original quality"),
        ], Color::Green),
    ];

    let mut all_items = Vec::new();
    for (title, rows, color) in &sections {
        all_items.push(ListItem::new(Line::from(vec![
            Span::styled(
                format!("  {} ", title),
                Style::default().fg(*color).add_modifier(Modifier::BOLD),
            ),
        ])));
        for (key, desc) in rows {
            all_items.push(ListItem::new(Line::from(vec![
                Span::styled(
                    format!("    {:<24}", key),
                    Style::default().fg(Color::White).add_modifier(Modifier::BOLD),
                ),
                Span::styled(*desc, Style::default().fg(Color::DarkGray)),
            ])));
        }
        all_items.push(ListItem::new(Line::from("")));
    }

    let list = List::new(all_items).block(block);
    f.render_widget(list, card);
}

// ─── summary bar ────────────────────────────────────────────────────────────

fn render_summary_bar(f: &mut Frame, area: Rect, app: &App) {
    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Cyan));

    let state = app.queue_state.as_ref();
    let completed = state.map(|s| s.completed).unwrap_or(0);
    let failed = state.map(|s| s.failed).unwrap_or(0);
    let running = state.map(|s| s.running).unwrap_or(0);
    let pending = state.map(|s| s.pending).unwrap_or(0);
    let skipped = state.map(|s| s.tasks.iter().filter(|t| t.status == crush_core::core::queue::TaskStatus::Skipped).count()).unwrap_or(0);
    let total = state.map(|s| s.total).unwrap_or(0);

    let (status, status_c) = if app.processing {
        ("▶ PROCESSING", Color::Green)
    } else if completed + failed + skipped > 0 {
        ("✓ DONE", Color::Green)
    } else {
        ("○ IDLE", Color::DarkGray)
    };

    let elapsed = if app.elapsed_ms > 0 {
        format!(" {:.1}s ", app.elapsed_ms as f64 / 1000.0)
    } else {
        String::new()
    };

    let hint = match app.active_panel {
        Panel::Menu => "↑↓ navigate  Enter select  H help  q quit",
        Panel::Files => if app.select_mode { "↑↓ navigate  Space toggle  Enter confirm" } else { "↑↓ navigate  Click select  Tab back" },
        Panel::Quality => "↑↓ select  0-9 custom  L lossless  Tab back",
        Panel::Format => "↑↓ select  Enter confirm  Tab back",
        Panel::Queue => "c cancel  x clear  Esc back",
        Panel::QualitySubmenu => "↑↓ select  Enter confirm  L lossless  Esc back",
        Panel::BackupConfirm => "↑↓ select  Enter confirm  Esc back",
        Panel::Help => "Esc or H to close",
    };

    let mut spans = vec![
        Span::styled(format!(" {} ", status), Style::default().fg(status_c).add_modifier(Modifier::BOLD)),
        Span::styled("│ ", Style::default().fg(Color::DarkGray)),
    ];

    if total > 0 {
        let bar_w = 14;
        let done = completed + failed + skipped;
        let bar = pct(done as f64 / total as f64, bar_w);
        spans.push(Span::styled(bar, Style::default().fg(Color::Cyan)));
        spans.push(Span::styled(format!(" {}/{} ", done, total), Style::default().fg(Color::White)));
    }

    spans.push(Span::styled(format!("{} OK ", completed), Style::default().fg(Color::Green)));
    if failed > 0 {
        spans.push(Span::styled(format!("{} FAIL ", failed), Style::default().fg(Color::Red)));
    }
    if skipped > 0 {
        spans.push(Span::styled(format!("{} SKIP ", skipped), Style::default().fg(Color::Yellow)));
    }
    if running > 0 {
        spans.push(Span::styled(format!("{} RUN ", running), Style::default().fg(Color::Cyan)));
    }
    if pending > 0 {
        spans.push(Span::styled(format!("{} WAIT ", pending), Style::default().fg(Color::DarkGray)));
    }

    if !elapsed.is_empty() {
        spans.push(Span::styled("│ ", Style::default().fg(Color::DarkGray)));
        spans.push(Span::styled(elapsed, Style::default().fg(Color::Cyan)));
    }

    spans.push(Span::styled("│ ", Style::default().fg(Color::DarkGray)));
    spans.push(Span::styled(&app.status_msg, Style::default().fg(Color::DarkGray)));
    spans.push(Span::styled("  │  ", Style::default().fg(Color::DarkGray)));
    spans.push(Span::styled(hint, Style::default().fg(Color::DarkGray)));

    f.render_widget(Paragraph::new(Line::from(spans)).block(block), area);
}
