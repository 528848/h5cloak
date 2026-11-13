UPDATE attribution_params
SET status = 'enabled', remark = '活动码，须在登记列表内'
WHERE name = 'cid' AND param_type = 'campaign';

UPDATE attribution_params
SET remark = '分销码，须在登记列表内'
WHERE name = 'ref' AND param_type = 'distributor';
