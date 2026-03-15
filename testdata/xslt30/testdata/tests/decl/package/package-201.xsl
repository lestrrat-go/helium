<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:test="http://example.com/xslt/package/test"
                exclude-result-prefixes="test xs"
                version="3.0">

  <xsl:template name="xsl:initial-template">
    <xsl:call-template name="test:main"/>
  </xsl:template>

  <!-- Invalid package version number ('1.0.0' instead of 1.0.0) should
       raise a package-not-found error. -->
  <xsl:use-package name="http://www.w3.org/xslt30tests/package-200"
                   package-version="'1.0.0'"/>

</xsl:stylesheet>
