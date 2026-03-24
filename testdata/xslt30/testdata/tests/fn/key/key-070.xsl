<?xml version="1.0" encoding="UTF-8"?>
<?spec xslt#keys?>
<!-- test xsl:key with type=xs:double using NaN as a key value -->

<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">

    <xsl:key name="i" match="item" use="xs:double(.)"/>

    <xsl:template match="/">
        <out>
            <xsl:value-of select="key('i', xs:double('NaN'))/@id" separator=","/>
        </out>
    </xsl:template>
</xsl:stylesheet>
