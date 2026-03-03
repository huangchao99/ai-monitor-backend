INSERT INTO cameras (name, rtsp_url, location) VALUES ('cam01',  'rtsp://admin:hifleet321@192.168.254.124:554/Streaming/Channels/101', '临港办公室');
INSERT INTO algorithms (algo_key, algo_name, category) VALUES
('no_person', '离岗', '行为分析'),('smoking', '吸烟', '行为分析'),
('play_phone', '打电话', '行为分析'),('eye_close', '闭眼', '行为分析'),('yawning', '打哈欠', '行为分析'),
('eat_banana', '吃香蕉', '行为分析');
INSERT INTO tasks (task_name, camera_id, status) VALUES ('测试任务01', 1, 1);
INSERT INTO task_algo_details (task_id, algo_id, roi_config, algo_params, alarm_config)
VALUES (
    1,
    3, -- 对应玩手机检测算法
    '[[0.1, 0.1], [0.5, 0.1], [0.5, 0.4], [0.1, 0.4]]',   -- 归一化的区域坐标数组，'[]'空数组表示全屏
    '{"skip_frame": 10, "confidence": 0.35, "duration": 30, }', -- 玩手机检测算法需要的参数
--skip_frame,表示10帧检测一帧，confidence表示检测置信度需要大于0.35，duration表示需要持续检测到30s才算玩手机
    '{"alarm_interval": 60 }' --报警参数，冷却时间报警间隔60s
);

INSERT INTO task_algo_details (task_id, algo_id, roi_config, algo_params, alarm_config)
VALUES (
    1,
    1, -- 对应离岗检测算法
    '[]',   -- 归一化的区域坐标数组，'[]'空数组表示全屏
    '{"skip_frame": 10, "confidence": 0.35, "duration": 120, }', -- 离岗检测算法需要的参数
--skip_frame,表示10帧检测一帧，confidence表示检测置信度需要大于0.35，duration表示需要持续120s没检测到人才算离岗
    '{"alarm_interval": 60 }' --报警参数，冷却时间，报警间隔60s
);

INSERT INTO task_algo_details (task_id, algo_id, roi_config, algo_params, alarm_config)
VALUES (
    1,
    4, -- 对应吃香蕉检测算法
    '[]',   -- 归一化的区域坐标数组，'[]'空数组表示全屏
    '{"skip_frame": 10, "confidence": 0.35, "duration": 30, }',
    '{"alarm_interval": 60 }'
);

INSERT INTO task_algo_details (task_id, algo_id, roi_config, algo_params, alarm_config)
VALUES (
    1,
    6, -- 对应打哈欠检测算法
    '[]',   -- 归一化的区域坐标数组，'[]'空数组表示全屏
    '{"skip_frame": 10, "confidence": 0.35, "yawn_count": 3,"yawn_duration":180 }', -- 打哈欠检测算法需要的参数
--skip_frame,表示10帧检测一帧，confidence表示检测置信度需要大于0.35，yawn_count表示打哈欠次数，yawn_duration表示判定时间范围，即180s内打3个哈欠才报警
    '{"alarm_interval": 60 }' --报警参数，冷却时间报警间隔60s
);
