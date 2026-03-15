<xsl:package
    name="http://www.w3.org/xslt30tests/accept-913"  
    package-version="1.0.0"
    version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <!-- tests error XTDE0040 to be raised on abstract initial template (instead of XTDE3052) -->

    <xsl:use-package name="http://www.w3.org/xslt30tests/package-913" package-version="1.0.0">
        <!-- not specifically accepting xsl:initial-template makes it private -->
    </xsl:use-package>  

</xsl:package>    