<xsl:package
  name="http://www.w3.org/xslt30tests/package-915"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <!-- tests error XTSE0020 to be raised on invalid token -->

    <xsl:use-package name="http://www.w3.org/xslt30tests/package-915" package-version="1.0.0">
        <xsl:accept names="*:m2 #unnamed" visibility="private" component="*" />
    </xsl:use-package>  

    <xsl:template name="xsl:initial-template">
      <not-ok/>
    </xsl:template>

</xsl:package>    