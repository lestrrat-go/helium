<xsl:package name="http://www.w3.org/xslt30tests/use-package-212" package-version="1.0.0"
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- tests that a given package version is chosen from a zero-truncated version -->

  <xsl:use-package name="http://www.w3.org/xslt30tests/use-package-base-004" package-version="2.0"/>

  <xsl:template name="main" visibility="public">
    <package-version>
      <xsl:value-of select="$package-version"/>
    </package-version>
  </xsl:template>

</xsl:package>
