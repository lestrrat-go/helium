<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
                
<!-- test that "mod" can be used as a variable name --> 
<?spec xslt#local-variables?> 
<!-- See Saxon bug 6.3/008 -->               
  
  <xsl:template match="/">
  <out changed="2004-03-05">
    <xsl:apply-templates select="*">
      <xsl:with-param name="mod" select="*"/>
    </xsl:apply-templates>
  </out>
  </xsl:template>

  <xsl:template match="*">
    <xsl:param name="mod"/>
    <xsl:apply-templates>
      <xsl:with-param name="mod" select="$mod"/>
    </xsl:apply-templates>
  </xsl:template>
  
</xsl:stylesheet>
