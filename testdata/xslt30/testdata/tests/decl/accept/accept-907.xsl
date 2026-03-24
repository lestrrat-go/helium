<xsl:package
  name="http://www.w3.org/xslt30tests/accept-907"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <!-- tests error XTDE3052 to be raised on abstract component -->

    <xsl:use-package name="http://www.w3.org/xslt30tests/package-905" package-version="1.0.0">
        <xsl:accept names="xsl:initial-template" visibility="public" component="template" />
        <!-- if a wildcard has an invalid combination (abstract -> public) it is treated as a non-match, and abstract gets default: hidden -->
        <xsl:accept names="*" visibility="public" component="*" />
    </xsl:use-package>  

</xsl:package>    