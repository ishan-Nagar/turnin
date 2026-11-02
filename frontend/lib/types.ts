export type Role = "student" | "professor";

export interface User { id: string; name: string; email: string; role: Role }
export interface Course { id: string; title: string; professor_id: string }

export interface Assignment {
  id: string;
  course_id: string;
  title: string;
  description: string;
  deadline: string;
  max_score: number;
  course?: Course;
  submission_count?: number;          // professors
  submission?: Submission | null;     // students
}

export interface Submission {
  id: string;
  assignment_id: string;
  student_id: string;
  file_url: string;
  status: string;
  score: number | null;
  submitted_at: string;
  student?: User;
  assignment?: Assignment;
  /** Client-only flag: true while the write has not been confirmed by the API. */
  optimistic?: boolean;
}

export interface Session { token: string; user: User }
