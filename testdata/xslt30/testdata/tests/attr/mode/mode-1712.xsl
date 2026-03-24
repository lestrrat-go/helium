<xsl:package
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
  default-mode="a">
  
  <xsl:mode name="a" visibility="public"/>
  <xsl:mode/>  
  
  <xsl:template match="/">
    <out xsl:default-mode="#unnamed">
      <xsl:apply-templates select="doc/a"/>      
    </out>
  </xsl:template>

  <xsl:template match="a" mode="a">
    <xsl:text>mode-a:</xsl:text>
    <xsl:value-of select="."/>
  </xsl:template>  
  
  <xsl:template match="a" mode="#unnamed">
    <xsl:text>no-mode:</xsl:text>
    <xsl:value-of select="."/>
  </xsl:template>

</xsl:package>   