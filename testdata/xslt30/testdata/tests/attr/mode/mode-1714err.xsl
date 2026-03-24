<xsl:package
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <!-- private by default -->
  <xsl:mode name="a"/>
  <xsl:mode/>  
  
  <!-- using default-mode on a non-root XSLT element does not make it eligible as initial mode, XTDE0045 -->
  <xsl:template match="/" default-mode="a">
    <out>
      <xsl:apply-templates select="doc/a"/>      
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