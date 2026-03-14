<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:f="http://f.com/" exclude-result-prefixes="f xs"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">

    <!-- test value range of unsignedLong -->

    <xsl:template match="/">
        <a val="{xs:unsignedLong('0'), xs:unsignedLong('999999999999'), xs:unsignedLong('18446744073709551615'),
xs:unsignedLong('18446744073709551615') + 3.14159}">OK</a>
    </xsl:template>
</xsl:stylesheet>
