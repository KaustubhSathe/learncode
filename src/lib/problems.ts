export async function deleteProblem(id: number) {
  const { error } = await supabase
    .from('problems')
    .update({ deleted_at: new Date().toISOString() })
    .eq('id', id);
  
  if (error) throw error;
} 