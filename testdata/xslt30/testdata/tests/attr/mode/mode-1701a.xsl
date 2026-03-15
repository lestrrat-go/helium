<xsl:package
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
  default-mode="a">
  
  <!-- tests invocation of non-public mode, error scenario -->
  
  <xsl:mode name="a" />
  
  <xsl:template match="/">
    <ok/>
  </xsl:template>

</xsl:package>   