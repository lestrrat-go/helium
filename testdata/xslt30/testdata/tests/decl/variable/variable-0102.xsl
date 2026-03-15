<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
                
<!-- XSLT 2.0 allows two local variables to have the same name -->
<?spec xslt#local-variables?> 
           
  
  <xsl:template match="/">
    <xsl:apply-templates select="*">
      <xsl:with-param name="mod" select="3"/>
    </xsl:apply-templates>
  </xsl:template>

  <xsl:template match="*">
    <xsl:param name="mod"/>
    <xsl:variable name="mod" select="$mod + 4"/>
    <out result="{$mod}"/>
  </xsl:template>
  
</xsl:stylesheet>
