<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-lower-case?>
  <!-- PURPOSE -->
  <!-- Test lower-case() applied to an empty sequence -->
  
  <xsl:template match="doc">
    <out>
      <xsl:variable name="x" select="lower-case(())"/>
        <z len="{string-length($x)}" count="{count($x)}"/>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
