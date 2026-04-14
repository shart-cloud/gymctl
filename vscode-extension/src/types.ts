export type WorkspaceMode = "gymctl" | "coder-lab" | "none";

export interface DetectionResult {
  mode: WorkspaceMode;
  tasksDir?: string;
  specPath?: string;
}

export interface ListExercise {
  name: string;
  title: string;
  status: "completed" | "in-progress" | "not-started";
  difficulty: string;
  locked: boolean;
}

export interface ListResponse {
  exercises: ListExercise[];
  summary: {
    total: number;
    completed: number;
    inProgress: number;
    notStarted: number;
    totalPoints: number;
    earnedPoints: number;
  };
}

export interface StatusResponse {
  current: string | null;
  summary: {
    total: number;
    completed: number;
    inProgress: number;
    notStarted: number;
    totalPoints: number;
    earnedPoints: number;
    completionPercent: number;
  };
}

export interface CheckResult {
  name: string;
  passed: boolean;
  message: string;
}

export interface MCQResult {
  id: string;
  passed: boolean;
  message?: string;
}

export interface CheckResponse {
  exercise: string;
  allPassed: boolean;
  passedCount: number;
  totalCount: number;
  score: number;
  pointsAvailable: number;
  checks: CheckResult[];
  mcqs?: MCQResult[];
}

export interface HintResponse {
  hintIndex: number | null;
  content: string | null;
  hintsRemaining: number;
  nextHintCost: number;
}

export interface LabSection {
  type: "markdown" | "mcq";
  markdown?: string;
  mcq?: {
    id: string;
    question?: string;
    options: Array<{
      letter: string;
      text: string;
      selected: boolean;
    }>;
    selected?: string[];
  };
}

export interface DescribeResponse {
  name: string;
  title: string;
  track: string;
  week?: number;
  order?: number;
  difficulty: string;
  estimatedTime?: string;
  points: number;
  description?: string;
  learningOutcomes?: string[];
  tags?: string[];
  prerequisites?: string[];
  labPath?: string;
  labSections?: LabSection[];
}
