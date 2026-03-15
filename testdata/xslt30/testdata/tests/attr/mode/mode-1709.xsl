<xsl:package
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <xsl:mode name="a"/>
  <xsl:mode/>  
  
  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="doc/a" default-mode="a"/>      
    </out>
  </xsl:template>

  <xsl:template match="a" mode="a">
    <xsl:text>mode-a:</xsl:text>
    <xsl:value-of select="."/>
  </xsl:template>  
  
  <xsl:template match="a">
    <xsl:text>no-mode:</xsl:text>
    <xsl:value-of select="."/>
  </xsl:template>

</xsl:package>   