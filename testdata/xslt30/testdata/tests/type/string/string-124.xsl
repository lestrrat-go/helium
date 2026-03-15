<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-string?>
  <!-- PURPOSE -->
  <!-- Test string() with zero arguments: returns the string value of the context node -->
  
  <xsl:template match="doc">
    <out>
      <xsl:value-of select='string()'/>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
