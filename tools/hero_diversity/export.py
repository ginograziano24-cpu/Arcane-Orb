import bpy,os,json,math,time
BASE=os.path.dirname(os.path.abspath(__file__));OUT=os.path.join(BASE,'models');os.makedirs(OUT,exist_ok=True);results=[]
for cid in ['mirabel','fumbleton','prism_colossus','crucible_golem','crystal_warden','guardian','berserker','arcane','engineer','lucky','pyro','storm']:
 if not os.path.exists(os.path.join(BASE,'characters',cid+'.blend')):continue
 bpy.ops.wm.open_mainfile(filepath=os.path.join(BASE,'characters',cid+'.blend'));sc=bpy.context.scene;sc.frame_set(1)
 rig=next(o for o in sc.objects if o.type=='ARMATURE');rig.animation_data_clear()
 for pb in rig.pose.bones:pb.rotation_mode='XYZ';pb.rotation_euler=(0,0,0);pb.scale=(1,1,1)
 bpy.context.view_layer.update()
 def volume_only(ob):
  mats=[slot.material for slot in ob.material_slots if slot.material]
  if not mats:return False
  return all(m.use_nodes and any(n.type=='OUTPUT_MATERIAL' and n.inputs['Volume'].is_linked and not n.inputs['Surface'].is_linked for n in m.node_tree.nodes) for m in mats)
 meshes=[o for o in sc.objects if o.type=='MESH' and not o.hide_render and not o.name.startswith('STUDIO') and not volume_only(o)]
 excluded=[o.name for o in sc.objects if o.type=='MESH' and o not in meshes]
 before=sum(len(o.data.vertices) for o in meshes)
 for o in meshes:
  bpy.context.view_layer.objects.active=o
  for m in list(o.modifiers):
   if m.type not in ['ARMATURE','SUBSURF']:
    try:bpy.ops.object.modifier_apply(modifier=m.name)
    except:pass
   elif m.type=='SUBSURF':o.modifiers.remove(m)
 bpy.ops.object.select_all(action='DESELECT')
 for o in meshes:o.select_set(True)
 active=next((o for o in meshes if any(m.type=='ARMATURE' for m in o.modifiers)),meshes[0]);bpy.context.view_layer.objects.active=active;bpy.ops.object.join();mesh=bpy.context.object;mesh.name=cid+'_cinematic_mesh'
 if not any(m.type=='ARMATURE' for m in mesh.modifiers):m=mesh.modifiers.new('Rig','ARMATURE');m.object=rig
 # Keep the high quality Blender source intact; only the realtime export is consolidated.
 flat_colors={}
 for mat in mesh.data.materials:
  if mat and mat.use_nodes:
   for n in mat.node_tree.nodes:
    if n.type=='BSDF_PRINCIPLED' and n.inputs['Base Color'].is_linked:
     flat_colors[mat.name]=list(n.inputs['Base Color'].default_value)
     for link in list(n.inputs['Base Color'].links):mat.node_tree.links.remove(link)
    if n.type=='BSDF_PRINCIPLED' and n.inputs['Roughness'].is_linked:
     for link in list(n.inputs['Roughness'].links):mat.node_tree.links.remove(link)
 bpy.ops.object.select_all(action='DESELECT');mesh.select_set(True);rig.select_set(True);bpy.context.view_layer.objects.active=mesh
 path=os.path.join(OUT,cid+'.glb')
 bpy.ops.export_scene.gltf(filepath=path,export_format='GLB',use_selection=True,export_animations=False,export_apply=False,export_cameras=False,export_lights=False,export_yup=True)
 results.append({'id':cid,'sourceVertices':before,'exportVertices':len(mesh.data.vertices),'bytes':os.path.getsize(path),'sourcePreserved':True,'flattenedProceduralBaseColors':flat_colors,'excludedHiddenOrVolumeMeshes':excluded})
 with open(os.path.join(OUT,'export-check.json'),'w') as f:json.dump(results,f,indent=2)
 print('EXPORTED',cid,os.path.getsize(path),flush=True)
