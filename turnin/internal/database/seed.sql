-- turnin seed data: 2 professors, 10 students, 3 courses, 5 assignments, 20 submissions.
-- Idempotent (INSERT OR IGNORE). Requires the schema to exist: boot the API once, or use `go run ./cmd/api --seed`.
-- Every account's password is: password   (bcrypt cost 10)

-- ---------------------------------------------------------------- users
INSERT OR IGNORE INTO users (id, name, email, password_hash, role) VALUES
('a0000000-0000-4000-8000-000000000001', 'Dr. Meera Iyer',      'meera.iyer@faculty.nalanda.ac.in',      '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'professor'),
('a0000000-0000-4000-8000-000000000002', 'Prof. Vikram Malhotra','vikram.malhotra@faculty.nalanda.ac.in','$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'professor'),
('b0000000-0000-4000-8000-000000000001', 'Aarav Sharma',  'aarav.sharma@students.nalanda.ac.in',  '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'student'),
('b0000000-0000-4000-8000-000000000002', 'Riya Patel',    'riya.patel@students.nalanda.ac.in',    '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'student'),
('b0000000-0000-4000-8000-000000000003', 'Vihaan Gupta',  'vihaan.gupta@students.nalanda.ac.in',  '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'student'),
('b0000000-0000-4000-8000-000000000004', 'Ananya Reddy',  'ananya.reddy@students.nalanda.ac.in',  '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'student'),
('b0000000-0000-4000-8000-000000000005', 'Arjun Nair',    'arjun.nair@students.nalanda.ac.in',    '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'student'),
('b0000000-0000-4000-8000-000000000006', 'Diya Mehta',    'diya.mehta@students.nalanda.ac.in',    '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'student'),
('b0000000-0000-4000-8000-000000000007', 'Kabir Singh',   'kabir.singh@students.nalanda.ac.in',   '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'student'),
('b0000000-0000-4000-8000-000000000008', 'Ishita Verma',  'ishita.verma@students.nalanda.ac.in',  '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'student'),
('b0000000-0000-4000-8000-000000000009', 'Rohan Desai',   'rohan.desai@students.nalanda.ac.in',   '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'student'),
('b0000000-0000-4000-8000-000000000010', 'Saanvi Joshi',  'saanvi.joshi@students.nalanda.ac.in',  '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'student');

-- -------------------------------------------------------------- courses
INSERT OR IGNORE INTO courses (id, title, professor_id) VALUES
('c0000000-0000-4000-8000-000000000001', 'CS301: Data Structures & Algorithms', 'a0000000-0000-4000-8000-000000000001'),
('c0000000-0000-4000-8000-000000000002', 'CS342: Database Management Systems',  'a0000000-0000-4000-8000-000000000001'),
('c0000000-0000-4000-8000-000000000003', 'CS415: Distributed Systems',          'a0000000-0000-4000-8000-000000000002');

-- ---------------------------------------------------------- assignments
INSERT OR IGNORE INTO assignments (id, course_id, title, description, deadline, max_score) VALUES
('d0000000-0000-4000-8000-000000000001', 'c0000000-0000-4000-8000-000000000001',
 'Red-Black Tree Implementation',
 'Implement insertion, deletion and rebalancing for a red-black tree in C++17. Include an invariant checker and a stress test of 10^6 random operations.',
 datetime('now', '+3 days', '+4 hours'), 100),
('d0000000-0000-4000-8000-000000000002', 'c0000000-0000-4000-8000-000000000001',
 'Graph Algorithms Problem Set',
 'Solve problems 1-8: Dijkstra with a custom binary heap, Kosaraju SCC, and Floyd-Warshall path reconstruction. Submit a single PDF with proofs.',
 datetime('now', '-6 days'), 50),
('d0000000-0000-4000-8000-000000000003', 'c0000000-0000-4000-8000-000000000002',
 'Query Optimizer Report',
 'Analyse EXPLAIN plans for the provided TPC-H workload and propose three index or rewrite optimisations with measured speed-ups.',
 datetime('now', '+6 days', '-3 hours'), 100),
('d0000000-0000-4000-8000-000000000004', 'c0000000-0000-4000-8000-000000000002',
 'B+ Tree Index Benchmark',
 'Benchmark point and range lookups on a B+ tree versus a hash index over 1M rows. Report p50/p95/p99 latencies.',
 datetime('now', '-2 days'), 40),
('d0000000-0000-4000-8000-000000000005', 'c0000000-0000-4000-8000-000000000003',
 'Raft Consensus Lab',
 'Implement leader election and log replication in Go. Your implementation must pass the supplied network-partition and crash-recovery test suite.',
 datetime('now', '+10 days', '+1 hours'), 150);

-- ---------------------------------------------------------- submissions
INSERT OR IGNORE INTO submissions (id, assignment_id, student_id, file_url, status, score, submitted_at) VALUES
-- Graph Algorithms Problem Set (closed, graded)
('e0000000-0000-4000-8000-000000000001','d0000000-0000-4000-8000-000000000002','b0000000-0000-4000-8000-000000000001','/uploads/seed/aarav_sharma_graph_algorithms.pdf','Graded',46,datetime('now','-6 days','-5 hours','-12 minutes')),
('e0000000-0000-4000-8000-000000000002','d0000000-0000-4000-8000-000000000002','b0000000-0000-4000-8000-000000000002','/uploads/seed/riya_patel_graph_algorithms.pdf','Graded',48,datetime('now','-6 days','-9 hours','-41 minutes')),
('e0000000-0000-4000-8000-000000000003','d0000000-0000-4000-8000-000000000002','b0000000-0000-4000-8000-000000000003','/uploads/seed/vihaan_gupta_graph_algorithms.pdf','Graded',41,datetime('now','-6 days','-1 hours','-20 minutes')),
('e0000000-0000-4000-8000-000000000004','d0000000-0000-4000-8000-000000000002','b0000000-0000-4000-8000-000000000004','/uploads/seed/ananya_reddy_graph_algorithms.pdf','Graded',50,datetime('now','-7 days','-3 hours','-7 minutes')),
('e0000000-0000-4000-8000-000000000005','d0000000-0000-4000-8000-000000000002','b0000000-0000-4000-8000-000000000005','/uploads/seed/arjun_nair_graph_algorithms.pdf','Graded',37,datetime('now','-6 days','-2 hours','-45 minutes')),
('e0000000-0000-4000-8000-000000000006','d0000000-0000-4000-8000-000000000002','b0000000-0000-4000-8000-000000000006','/uploads/seed/diya_mehta_graph_algorithms.pdf','Graded',44,datetime('now','-6 days','-14 hours','-3 minutes')),
-- B+ Tree Index Benchmark (closed, partially graded)
('e0000000-0000-4000-8000-000000000007','d0000000-0000-4000-8000-000000000004','b0000000-0000-4000-8000-000000000001','/uploads/seed/aarav_sharma_bplus_benchmark.zip','Graded',35,datetime('now','-2 days','-6 hours','-18 minutes')),
('e0000000-0000-4000-8000-000000000008','d0000000-0000-4000-8000-000000000004','b0000000-0000-4000-8000-000000000002','/uploads/seed/riya_patel_bplus_benchmark.zip','Graded',38,datetime('now','-2 days','-11 hours','-30 minutes')),
('e0000000-0000-4000-8000-000000000009','d0000000-0000-4000-8000-000000000004','b0000000-0000-4000-8000-000000000003','/uploads/seed/vihaan_gupta_bplus_benchmark.zip','Scanned: 0% Plagiarism',NULL,datetime('now','-2 days','-1 hours','-5 minutes')),
('e0000000-0000-4000-8000-000000000010','d0000000-0000-4000-8000-000000000004','b0000000-0000-4000-8000-000000000005','/uploads/seed/arjun_nair_bplus_benchmark.zip','Scanned: 0% Plagiarism',NULL,datetime('now','-3 days','-2 hours','-9 minutes')),
('e0000000-0000-4000-8000-000000000011','d0000000-0000-4000-8000-000000000004','b0000000-0000-4000-8000-000000000007','/uploads/seed/kabir_singh_bplus_benchmark.zip','Scanned: 0% Plagiarism',NULL,datetime('now','-2 days','-20 minutes')),
-- Red-Black Tree Implementation (open)
('e0000000-0000-4000-8000-000000000012','d0000000-0000-4000-8000-000000000001','b0000000-0000-4000-8000-000000000002','/uploads/seed/riya_patel_rbtree.zip','Scanned: 0% Plagiarism',NULL,datetime('now','-1 days','-3 hours','-12 minutes')),
('e0000000-0000-4000-8000-000000000013','d0000000-0000-4000-8000-000000000001','b0000000-0000-4000-8000-000000000004','/uploads/seed/ananya_reddy_rbtree.zip','Scanned: 0% Plagiarism',NULL,datetime('now','-20 hours','-4 minutes')),
('e0000000-0000-4000-8000-000000000014','d0000000-0000-4000-8000-000000000001','b0000000-0000-4000-8000-000000000006','/uploads/seed/diya_mehta_rbtree.zip','Scanned: 0% Plagiarism',NULL,datetime('now','-4 hours','-33 minutes')),
('e0000000-0000-4000-8000-000000000015','d0000000-0000-4000-8000-000000000001','b0000000-0000-4000-8000-000000000008','/uploads/seed/ishita_verma_rbtree.zip','Scanned: 0% Plagiarism',NULL,datetime('now','-42 minutes')),
('e0000000-0000-4000-8000-000000000016','d0000000-0000-4000-8000-000000000001','b0000000-0000-4000-8000-000000000010','/uploads/seed/saanvi_joshi_rbtree.zip','Pending',NULL,datetime('now','-3 minutes')),
-- Query Optimizer Report (open)
('e0000000-0000-4000-8000-000000000017','d0000000-0000-4000-8000-000000000003','b0000000-0000-4000-8000-000000000003','/uploads/seed/vihaan_gupta_query_optimizer.pdf','Scanned: 0% Plagiarism',NULL,datetime('now','-2 days','-5 hours','-27 minutes')),
('e0000000-0000-4000-8000-000000000018','d0000000-0000-4000-8000-000000000003','b0000000-0000-4000-8000-000000000005','/uploads/seed/arjun_nair_query_optimizer.pdf','Scanned: 0% Plagiarism',NULL,datetime('now','-9 hours','-15 minutes')),
('e0000000-0000-4000-8000-000000000019','d0000000-0000-4000-8000-000000000003','b0000000-0000-4000-8000-000000000009','/uploads/seed/rohan_desai_query_optimizer.pdf','Pending',NULL,datetime('now','-8 minutes')),
-- Raft Consensus Lab (open)
('e0000000-0000-4000-8000-000000000020','d0000000-0000-4000-8000-000000000005','b0000000-0000-4000-8000-000000000004','/uploads/seed/ananya_reddy_raft_lab.zip','Scanned: 0% Plagiarism',NULL,datetime('now','-5 hours','-50 minutes'));

