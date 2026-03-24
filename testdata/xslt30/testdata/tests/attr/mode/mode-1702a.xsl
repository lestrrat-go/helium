<xsl:package
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
  declared-modes="no"
  default-mode="a">
    
   <!-- tests invocation of non-public mode, error scenario -->

  <xsl:template match="/">
      <ok/>
      <!-- creating an implicit, named mode that is private -->
      <xsl:apply-templates select="*" mode="test" />
  </xsl:template>

</xsl:package>   