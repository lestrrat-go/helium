<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!--  set context item to a sequence of 2 items within xsl:evaluate -->


  <xsl:param name="e1">true()</xsl:param>
  <xsl:param name="e3">@x</xsl:param>
    
  <xsl:template match="/">
    <out>
      <a><xsl:evaluate xpath="($e3)" context-item="1 to (1 + count($e1))"/></a>
    </out>
      
  </xsl:template>
  
</xsl:stylesheet>
