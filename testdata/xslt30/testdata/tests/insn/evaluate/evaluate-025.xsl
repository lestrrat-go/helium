<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!--  call @x when context item is undefined within xsl:evaluate -->

<xsl:param name="e3">@x</xsl:param>
    
  <xsl:template match="/">
    <out>
      <a><xsl:evaluate xpath="($e3)" context-item="()"/></a>
    </out>
      
  </xsl:template>
  
</xsl:stylesheet>
