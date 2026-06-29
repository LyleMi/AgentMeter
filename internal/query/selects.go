package query

const sessionSelect = `SELECT
		s.id, s.source_id, src.root_path, src.sessions_path, s.source_file_id, src.kind, src.name, COALESCE(NULLIF(s.session_key, ''), s.codex_session_id), s.codex_session_id, s.project_path, s.model, s.model_provider, s.originator, s.thread_source,
		s.agent_nickname, s.agent_role, s.started_at, s.ended_at, s.wall_duration_ms, s.active_duration_ms, s.model_duration_ms,
		s.tool_duration_ms, s.idle_duration_ms, s.event_count, s.parse_status,
		COALESCE(tu.model, s.model), COALESCE(tu.input_tokens, 0), COALESCE(tu.cached_input_tokens, 0), COALESCE(tu.output_tokens, 0),
		COALESCE(tu.reasoning_output_tokens, 0), COALESCE(tu.context_compression_tokens, 0), COALESCE(tu.total_tokens, 0), COALESCE(tu.source, 'unknown'),
		(SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id) AS tool_call_count,
		sf.path, sf.scan_status, sf.error
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN source_files sf ON sf.id = s.source_file_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id`

const toolCallSelect = `SELECT
		tc.id, tc.session_id, src.id, src.root_path, src.sessions_path,
		tc.started_at, tc.ended_at, tc.duration_ms, tc.tool_name, tc.status, tc.input_summary, tc.output_summary, tc.error,
		tc.raw_event_id, tc.call_id, tc.raw_start_event_id, tc.raw_end_event_id,
		COALESCE(start_event.source_line, 0), COALESCE(end_event.source_line, 0),
		COALESCE(start_event.raw_type, ''), COALESCE(end_event.raw_type, ''),
		COALESCE(start_event.summary, ''), COALESCE(end_event.summary, ''),
		COALESCE(start_event.raw_json, ''), COALESCE(end_event.raw_json, ''),
		COALESCE(NULLIF(sess.session_key, ''), sess.codex_session_id), sess.codex_session_id, sess.project_path,
		src.kind, src.name, sf.path
	FROM tool_calls tc
	JOIN sessions sess ON sess.id = tc.session_id
	JOIN sources src ON src.id = sess.source_id
	JOIN source_files sf ON sf.id = sess.source_file_id
	LEFT JOIN events start_event ON start_event.id = CASE WHEN tc.raw_start_event_id != 0 THEN tc.raw_start_event_id ELSE tc.raw_event_id END
	LEFT JOIN events end_event ON end_event.id = tc.raw_end_event_id`

const auditFindingSelect = `SELECT
		af.id, af.session_id, src.id, src.root_path, src.sessions_path,
		af.tool_call_id, af.source_file_id, af.raw_event_id, af.source_line, af.timestamp,
		af.source, af.event_type, af.category, af.severity, af.rule_id, af.title, af.description, af.evidence,
		af.command, af.shell_family, af.platform, af.decision, af.created_at,
		COALESCE(NULLIF(sess.session_key, ''), sess.codex_session_id), sess.codex_session_id, sess.project_path,
		src.kind, src.name, sf.path
	FROM audit_findings af
	JOIN sessions sess ON sess.id = af.session_id
	JOIN sources src ON src.id = sess.source_id
	JOIN source_files sf ON sf.id = af.source_file_id`
