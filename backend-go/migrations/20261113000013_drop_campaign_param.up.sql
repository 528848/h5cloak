-- 参数字典不再使用活动码。cid 停用；分销码文案改为归因码。
UPDATE attribution_params
SET status = 'disabled', remark = '已取消的活动码参数'
WHERE param_type = 'campaign';

UPDATE attribution_params
SET remark = '归因码，须在登记列表内'
WHERE name = 'ref' AND param_type = 'distributor';
