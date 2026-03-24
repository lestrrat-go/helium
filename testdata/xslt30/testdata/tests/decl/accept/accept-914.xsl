<xsl:package
    name="http://www.w3.org/xslt30tests/accept-914"  
    package-version="1.0.0"
    version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <!-- tests error XTSE3080 (not XTDE3052 or XTDE0040) to be raised on abstract initial template -->

    <xsl:use-package name="http://www.w3.org/xslt30tests/package-914" package-version="1.0.0">
        <!-- valid wildcard visibility for abstract xsl:initial-template -->
        <xsl:accept names="*" visibility="abstract" component="*" />
    </xsl:use-package>  

</xsl:package>    