<xsl:package
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
  declared-modes="no"
  default-mode="a">
    
    <!-- default mode is implicitly private, but is still publicly invocable -->
    <xsl:mode />

  <xsl:template match="/" mode="#unnamed">
      <ok/>
  </xsl:template>

</xsl:package>   