<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-upper-case?>
  <!-- PURPOSE -->
  <!-- Test upper-case() applied to an empty sequence -->
  
  <xsl:template match="doc">
    <out>
      <xsl:variable name="x" select="upper-case(())"/>
        <z len="{string-length($x)}" count="{count($x)}"/>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
