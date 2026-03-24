<xsl:package name="http://www.w3.org/xslt30tests/override-m-018" package-version="1.0.0"
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-m-001" declared-modes="no"
  exclude-result-prefixes="xs p">

  <xsl:use-package name="http://www.w3.org/xslt30tests/override-base-m-001" package-version="1.0.0"/>

  
  <xsl:template match="*" mode="m3">
    <wrong/>
  </xsl:template>

  <xsl:template name="main" visibility="public">
    <out>12</out>
  </xsl:template>


</xsl:package>
