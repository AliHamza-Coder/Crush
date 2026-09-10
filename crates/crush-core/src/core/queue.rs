use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use tokio::sync::RwLock;

static TASK_COUNTER: AtomicU64 = AtomicU64::new(1);

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq, Eq)]
pub enum Engine {
    Ffmpeg,
    Native,
    OnnxAi,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub enum TaskType {
    VideoCompress { quality: u8, format: String },
    VideoConvert { target: String, quality: u8 },
    ImageCompress { quality: u8, format: String },
    ImageConvert { target: String, quality: u8 },
    ImageUpscale { scale: u8 },
    AudioExtract { format: String, quality: u8 },
    AudioConvert { format: String, quality: u8 },
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq, Eq)]
pub enum TaskStatus {
    Pending,
    Running,
    Completed,
    Failed,
    Skipped,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct Task {
    pub id: u64,
    pub input_path: String,
    pub output_path: String,
    pub file_name: String,
    pub file_size: u64,
    pub file_size_str: String,
    pub task_type: TaskType,
    pub status: TaskStatus,
    pub progress: f32,
    pub engine: Engine,
    pub error: Option<String>,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct QueueState {
    pub tasks: Vec<Task>,
    pub total: usize,
    pub completed: usize,
    pub failed: usize,
    pub running: usize,
    pub pending: usize,
    pub elapsed_ms: u64,
}

#[derive(Clone)]
pub struct TaskQueue {
    tasks: Arc<RwLock<Vec<Task>>>,
    start_time: Option<std::time::Instant>,
}

impl Default for TaskQueue {
    fn default() -> Self {
        Self::new()
    }
}

impl TaskQueue {
    pub fn new() -> Self {
        Self {
            tasks: Arc::new(RwLock::new(Vec::new())),
            start_time: None,
        }
    }

    #[allow(clippy::too_many_arguments)]
    pub async fn add_task(
        &self,
        input_path: String,
        output_path: String,
        file_name: String,
        file_size: u64,
        file_size_str: String,
        task_type: TaskType,
        engine: Engine,
    ) -> u64 {
        let id = TASK_COUNTER.fetch_add(1, Ordering::SeqCst);
        let task = Task {
            id,
            input_path,
            output_path,
            file_name,
            file_size,
            file_size_str,
            task_type,
            status: TaskStatus::Pending,
            progress: 0.0,
            engine,
            error: None,
        };
        self.tasks.write().await.push(task);
        id
    }

    pub async fn start(&mut self) {
        self.start_time = Some(std::time::Instant::now());
    }

    pub async fn update_status(&self, id: u64, status: TaskStatus) {
        let mut tasks = self.tasks.write().await;
        if let Some(task) = tasks.iter_mut().find(|t| t.id == id) {
            task.status = status;
        }
    }

    pub async fn update_progress(&self, id: u64, progress: f32) {
        let mut tasks = self.tasks.write().await;
        if let Some(task) = tasks.iter_mut().find(|t| t.id == id) {
            task.progress = progress;
        }
    }

    pub async fn set_error(&self, id: u64, error: String) {
        let mut tasks = self.tasks.write().await;
        if let Some(task) = tasks.iter_mut().find(|t| t.id == id) {
            task.status = TaskStatus::Failed;
            task.error = Some(error);
        }
    }

    pub async fn get_state(&self) -> QueueState {
        let tasks = self.tasks.read().await;
        let completed = tasks
            .iter()
            .filter(|t| t.status == TaskStatus::Completed)
            .count();
        let failed = tasks
            .iter()
            .filter(|t| t.status == TaskStatus::Failed)
            .count();
        let running = tasks
            .iter()
            .filter(|t| t.status == TaskStatus::Running)
            .count();
        let pending = tasks
            .iter()
            .filter(|t| t.status == TaskStatus::Pending)
            .count();

        let elapsed_ms = self
            .start_time
            .map(|t| t.elapsed().as_millis() as u64)
            .unwrap_or(0);

        QueueState {
            tasks: tasks.clone(),
            total: tasks.len(),
            completed,
            failed,
            running,
            pending,
            elapsed_ms,
        }
    }

    pub async fn clear(&self) {
        self.tasks.write().await.clear();
    }
}
