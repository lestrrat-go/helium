<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test of lang() function with 2 arguments -->
  <?spec fo#func-lang?>
  <?same-as-1.0 no?>

<xsl:template match="/">
	<out><xsl:value-of select="for $p in doc/p return lang('en', $p)"/></out>
</xsl:template>


</xsl:stylesheet>
