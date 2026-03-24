<xsl:package 
  name="http://www.w3.org/xslt30tests/override-base-901"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-001"
  exclude-result-prefixes="xs p">
  
  <xsl:key name="k" match="*" use="."/>
  
  <xsl:accumulator name="ac" initial-value="0">
    <xsl:accumulator-rule match="*" select="$value+1"/>
  </xsl:accumulator>
  
  <xsl:decimal-format name="d" grouping-separator="." decimal-separator=","/>
  
  <xsl:template name="k" visibility="public">
    <b><xsl:copy-of select="key('k', 'x')"/></b>
  </xsl:template>
  
  <xsl:template name="ac" visibility="public">
    <b><xsl:value-of select="accumulator-after('ac')"/></b>
  </xsl:template>
  
  <xsl:template name="d" visibility="public">
    <b><xsl:value-of select="format-number(123456.78, '0.000,00', 'd')"/></b>
  </xsl:template>
  
  
  
</xsl:package>   