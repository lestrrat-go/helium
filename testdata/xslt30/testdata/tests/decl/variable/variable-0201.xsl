<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:f="http://fxsl.sf.net/"
xmlns:test="my:test"
  exclude-result-prefixes="f xs test"
>
  <!-- basic test of tunnel parameters -->
  <?spec xslt#tunnel-params?>
  <?same-as-1.0 no?>
  
  <xsl:template match="/">
   <xsl:apply-templates>
     <xsl:with-param name="t1" select="17" tunnel="yes"/>
   </xsl:apply-templates>
  </xsl:template>
  
  <xsl:template match="b">
    <b><xsl:apply-templates/></b>
  </xsl:template>
  
  <xsl:template match="d">
    <xsl:param name="t1" tunnel="yes" select="23"/>
    <d><xsl:value-of select="$t1"/></d>
  </xsl:template>
    

</xsl:stylesheet>
